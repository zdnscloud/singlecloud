package handler

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type CronJobManager struct {
	DefaultHandler
	clusters *ClusterManager
}

func newCronJobManager(clusters *ClusterManager) *CronJobManager {
	return &CronJobManager{clusters: clusters}
}

func (m *CronJobManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	cronJob := ctx.Object.(*types.CronJob)
	err := createCronJob(cluster.KubeClient, namespace, cronJob)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate cronJob name %s", cronJob.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create cronJob failed %s", err.Error()))
		}
	}

	cronJob.SetID(cronJob.Name)
	return cronJob, nil
}

func (m *CronJobManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sCronJobs, err := getCronJobs(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("list cronJob info failed:%s", err.Error())
		}
		return nil
	}

	var cronJobs []*types.CronJob
	for _, item := range k8sCronJobs.Items {
		cronJobs = append(cronJobs, k8sCronJobToScCronJob(&item))
	}
	return cronJobs
}

func (m *CronJobManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	cronJob := ctx.Object.(*types.CronJob)
	k8sCronJob, err := getCronJob(cluster.KubeClient, namespace, cronJob.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get cronJob info failed:%s", err.Error())
		}
		return nil
	}

	return k8sCronJobToScCronJob(k8sCronJob)
}

func (m *CronJobManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	cronJob := ctx.Object.(*types.CronJob)
	if err := deleteCronJob(cluster.KubeClient, namespace, cronJob.GetID()); err != nil {
		if apierrors.IsNotFound(err) == false {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("cronJob %s with namespace %s desn't exist", cronJob.GetID(), namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete cronJob failed %s", err.Error()))
		}
	}

	return nil
}

func getCronJob(cli client.Client, namespace, name string) (*batchv1beta1.CronJob, error) {
	cronJob := batchv1beta1.CronJob{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &cronJob)
	return &cronJob, err
}

func getCronJobs(cli client.Client, namespace string) (*batchv1beta1.CronJobList, error) {
	cronJobs := batchv1beta1.CronJobList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &cronJobs)
	return &cronJobs, err
}

func createCronJob(cli client.Client, namespace string, cronJob *types.CronJob) error {
	k8sPodSpec, err := scContainersToK8sPodSpec(cronJob.Containers)
	if err != nil {
		return err
	}

	policy, err := scRestartPolicyToK8sRestartPolicy(cronJob.RestartPolicy)
	if err != nil {
		return err
	}

	k8sPodSpec.RestartPolicy = policy
	k8sCronJob := &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJob.Name,
			Namespace: namespace,
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule: cronJob.Schedule,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"cronjob-name": cronJob.Name}},
						Spec:       k8sPodSpec,
					},
				},
			},
		},
	}
	return cli.Create(context.TODO(), k8sCronJob)
}

func deleteCronJob(cli client.Client, namespace, name string) error {
	cronJob := &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), cronJob, client.PropagationPolicy(metav1.DeletePropagationForeground))
}

func k8sCronJobToScCronJob(k8sCronJob *batchv1beta1.CronJob) *types.CronJob {
	containers := k8sContainersToScContainers(k8sCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers,
		k8sCronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes)

	var objectReferences []types.ObjectReference
	for _, objectReference := range k8sCronJob.Status.Active {
		objectReferences = append(objectReferences, types.ObjectReference{
			Kind:            objectReference.Kind,
			Namespace:       objectReference.Namespace,
			Name:            objectReference.Name,
			UID:             string(objectReference.UID),
			APIVersion:      objectReference.APIVersion,
			ResourceVersion: objectReference.ResourceVersion,
			FieldPath:       objectReference.FieldPath,
		})
	}

	cronJobStatus := types.CronJobStatus{
		LastScheduleTime: k8sMetaV1TimePtrToISOTime(k8sCronJob.Status.LastScheduleTime),
		Active:           objectReferences,
	}

	cronJob := &types.CronJob{
		Name:          k8sCronJob.Name,
		Schedule:      k8sCronJob.Spec.Schedule,
		RestartPolicy: string(k8sCronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy),
		Containers:    containers,
		Status:        cronJobStatus,
	}
	cronJob.SetID(k8sCronJob.Name)
	cronJob.SetType(types.CronJobType)
	cronJob.SetCreationTimestamp(k8sCronJob.CreationTimestamp.Time)
	return cronJob
}
