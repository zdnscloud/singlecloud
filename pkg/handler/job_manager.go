package handler

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type JobManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newJobManager(clusters *ClusterManager) *JobManager {
	return &JobManager{clusters: clusters}
}

func (m *JobManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	job := ctx.Object.(*types.Job)
	err := createJob(cluster.KubeClient, namespace, job)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate job name %s", job.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create job failed %s", err.Error()))
		}
	}

	job.SetID(job.Name)
	return job, nil
}

func (m *JobManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sJobs, err := getJobs(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("list job info failed:%s", err.Error())
		}
		return nil
	}

	var jobs []*types.Job
	for _, item := range k8sJobs.Items {
		jobs = append(jobs, k8sJobToSCJob(&item))
	}
	return jobs
}

func (m *JobManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	job := ctx.Object.(*types.Job)
	k8sJob, err := getJob(cluster.KubeClient, namespace, job.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get job info failed:%s", err.Error())
		}
		return nil
	}

	return k8sJobToSCJob(k8sJob)
}

func (m *JobManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	job := ctx.Object.(*types.Job)
	if err := deleteJob(cluster.KubeClient, namespace, job.GetID()); err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("job %s with namespace %s desn't exist", job.GetID(), namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete job failed %s", err.Error()))
		}
	}

	return nil
}

func getJob(cli client.Client, namespace, name string) (*batchv1.Job, error) {
	job := batchv1.Job{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &job)
	return &job, err
}

func getJobs(cli client.Client, namespace string) (*batchv1.JobList, error) {
	jobs := batchv1.JobList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &jobs)
	return &jobs, err
}

func createJob(cli client.Client, namespace string, job *types.Job) error {
	k8sPodSpec, err := scContainersToK8sPodSpec(job.Containers)
	if err != nil {
		return err
	}

	policy, err := scRestartPolicyToK8sRestartPolicy(job.RestartPolicy)
	if err != nil {
		return err
	}

	k8sPodSpec.RestartPolicy = policy
	k8sJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.Name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"job-name": job.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"job-name": job.Name}},
				Spec:       k8sPodSpec,
			},
		},
	}
	return cli.Create(context.TODO(), k8sJob)
}

func deleteJob(cli client.Client, namespace, name string) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), job, client.PropagationPolicy(metav1.DeletePropagationForeground))
}

func k8sJobToSCJob(k8sJob *batchv1.Job) *types.Job {
	containers := k8sContainersToScContainers(k8sJob.Spec.Template.Spec.Containers, k8sJob.Spec.Template.Spec.Volumes)

	var conditions []types.JobCondition
	for _, condition := range k8sJob.Status.Conditions {
		conditions = append(conditions, types.JobCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastProbeTime:      resttypes.ISOTime(condition.LastProbeTime.Time),
			LastTransitionTime: resttypes.ISOTime(condition.LastTransitionTime.Time),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	jobStatus := types.JobStatus{
		StartTime:      k8sMetaV1TimePtrToISOTime(k8sJob.Status.StartTime),
		CompletionTime: k8sMetaV1TimePtrToISOTime(k8sJob.Status.CompletionTime),
		Active:         k8sJob.Status.Active,
		Succeeded:      k8sJob.Status.Succeeded,
		Failed:         k8sJob.Status.Failed,
		JobConditions:  conditions,
	}

	job := &types.Job{
		Name:          k8sJob.Name,
		RestartPolicy: string(k8sJob.Spec.Template.Spec.RestartPolicy),
		Containers:    containers,
		Status:        jobStatus,
	}
	job.SetID(k8sJob.Name)
	job.SetType(types.JobType)
	job.SetCreationTimestamp(k8sJob.CreationTimestamp.Time)
	return job
}

func k8sMetaV1TimePtrToISOTime(metav1Time *metav1.Time) (isoTime resttypes.ISOTime) {
	if metav1Time != nil {
		isoTime = resttypes.ISOTime(metav1Time.Time)
	}

	return isoTime
}
