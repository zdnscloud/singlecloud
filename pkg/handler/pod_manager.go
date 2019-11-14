package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	OwnerKindReplicaset  = "ReplicaSet"
	OwnerKindDeployment  = "Deployment"
	OwnerKindStatefulSet = "StatefulSet"
	OwnerKindDaemonSet   = "DaemonSet"
)

type PodManager struct {
	clusters *ClusterManager
}

func newPodManager(clusters *ClusterManager) *PodManager {
	return &PodManager{clusters: clusters}
}

func (m *PodManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetParent().GetID()
	ownerType := ctx.Resource.GetParent().GetType()
	ownerName := ctx.Resource.GetParent().GetID()
	k8sPods, err := getOwnerPods(cluster.KubeClient, namespace, ownerType, ownerName)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get pod info failed:%s", err.Error())
		}
		return nil
	}

	var pods []*types.Pod
	for _, k8sPod := range k8sPods.Items {
		pods = append(pods, k8sPodToSCPod(&k8sPod))
	}
	return pods
}

func (m *PodManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetParent().GetID()
	pod := ctx.Resource.(*types.Pod)
	k8sPod, err := getPod(cluster.KubeClient, namespace, pod.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get pod info failed:%s", err.Error())
		}
		return nil
	}

	return k8sPodToSCPod(k8sPod)
}

func (m *PodManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetParent().GetID()
	pod := ctx.Resource.(*types.Pod)
	err := deletePod(cluster.KubeClient, namespace, pod.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("pod %s desn't exist", pod.GetID()))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete pod failed %s", err.Error()))
		}
	}
	return nil
}

func getOwnerPods(cli client.Client, namespace, ownerType, ownerName string) (*corev1.PodList, error) {
	selector, err := getPodParentSelector(cli, namespace, ownerType, ownerName)
	if err != nil {
		return nil, err
	}

	if selector == nil {
		return nil, fmt.Errorf("%s %s has no selector", ownerType, ownerName)
	}

	k8sPods, err := getPods(cli, namespace, selector)
	if err != nil {
		return nil, err
	}

	filterPodBasedOnOwner(k8sPods, ownerType, ownerName)
	return k8sPods, nil
}

func getPodParentSelector(cli client.Client, namespace string, typ string, name string) (labels.Selector, error) {
	var selector *metav1.LabelSelector
	switch typ {
	case types.ResourceTypeDeployment:
		k8sDeploy, err := getDeployment(cli, namespace, name)
		if err != nil {
			return nil, err
		}

		selector = k8sDeploy.Spec.Selector
	case types.ResourceTypeDaemonSet:
		k8sDaemonSet, err := getDaemonSet(cli, namespace, name)
		if err != nil {
			return nil, err
		}

		selector = k8sDaemonSet.Spec.Selector
	case types.ResourceTypeStatefulSet:
		k8sStatefulSet, err := getStatefulSet(cli, namespace, name)
		if err != nil {
			return nil, err
		}

		selector = k8sStatefulSet.Spec.Selector
	case types.ResourceTypeJob:
		return genJobSelector(cli, namespace, name)
	case types.ResourceTypeCronJob:
		return genCronJobSelector(cli, namespace, name)
	default:
		return nil, fmt.Errorf("pod no such parent %v", typ)
	}

	if selector == nil {
		return nil, nil
	}

	return metav1.LabelSelectorAsSelector(selector)
}

func filterPodBasedOnOwner(pods *corev1.PodList, typ string, name string) {
	var results []corev1.Pod
	switch typ {
	case types.ResourceTypeDeployment:
		for _, pod := range pods.Items {
			rsHash, ok := pod.Labels["pod-template-hash"]
			if ok == false {
				continue
			}
			if len(pod.OwnerReferences) != 1 {
				continue
			}
			owner := pod.OwnerReferences[0]
			if owner.Kind == OwnerKindReplicaset && owner.Name == name+"-"+rsHash {
				results = append(results, pod)
			}
		}
	case types.ResourceTypeDaemonSet, types.ResourceTypeStatefulSet:
		kind := OwnerKindDaemonSet
		if typ == types.ResourceTypeStatefulSet {
			kind = OwnerKindStatefulSet
		}

		for _, pod := range pods.Items {
			if len(pod.OwnerReferences) != 1 {
				continue
			}
			owner := pod.OwnerReferences[0]
			if owner.Name == name && owner.Kind == kind {
				results = append(results, pod)
			}
		}
	case types.ResourceTypeJob, types.ResourceTypeCronJob:
		results = pods.Items
	}
	pods.Items = results
}

func getPod(cli client.Client, namespace, name string) (*corev1.Pod, error) {
	pod := corev1.Pod{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &pod)
	return &pod, err
}

func getPods(cli client.Client, namespace string, selector labels.Selector) (*corev1.PodList, error) {
	pods := corev1.PodList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace, LabelSelector: selector}, &pods)
	return &pods, err
}

func deletePod(cli client.Client, namespace, name string) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), pod)
}

func k8sPodToSCPod(k8sPod *corev1.Pod) *types.Pod {
	containers, _ := k8sPodSpecToScContainersAndVCTemplates(k8sPod.Spec.Containers, k8sPod.Spec.Volumes)

	var conditions []types.PodCondition
	for _, condition := range k8sPod.Status.Conditions {
		conditions = append(conditions, types.PodCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastProbeTime:      resource.ISOTime(condition.LastProbeTime.Time),
			LastTransitionTime: resource.ISOTime(condition.LastTransitionTime.Time),
		})
	}

	var statuses []types.ContainerStatus
	for _, status := range k8sPod.Status.ContainerStatuses {
		statuses = append(statuses, types.ContainerStatus{
			Name:         status.Name,
			Ready:        status.Ready,
			RestartCount: status.RestartCount,
			Image:        status.Image,
			ImageID:      status.ImageID,
			ContainerID:  status.ContainerID,
			LastState:    k8sContainerStateToScContainerState(status.LastTerminationState),
			State:        k8sContainerStateToScContainerState(status.State),
		})
	}

	podStatus := types.PodStatus{
		Phase:             string(k8sPod.Status.Phase),
		StartTime:         k8sMetaV1TimePtrToISOTime(k8sPod.Status.StartTime),
		HostIP:            k8sPod.Status.HostIP,
		PodIP:             k8sPod.Status.PodIP,
		PodConditions:     conditions,
		ContainerStatuses: statuses,
	}

	pod := &types.Pod{
		Name:       k8sPod.Name,
		NodeName:   k8sPod.Spec.NodeName,
		State:      helper.GetPodState(k8sPod),
		Containers: containers,
		Status:     podStatus,
	}
	pod.SetID(k8sPod.Name)
	pod.SetCreationTimestamp(k8sPod.CreationTimestamp.Time)
	return pod
}

func k8sContainerStateToScContainerState(k8sContainerState corev1.ContainerState) *types.ContainerState {
	var state *types.ContainerState
	if k8sContainerState.Waiting != nil {
		state = &types.ContainerState{
			Type:    types.WaitingState,
			Reason:  k8sContainerState.Waiting.Reason,
			Message: k8sContainerState.Waiting.Message,
		}
	} else if k8sContainerState.Running != nil {
		state = &types.ContainerState{
			Type:      types.RunningState,
			StartedAt: resource.ISOTime(k8sContainerState.Running.StartedAt.Time),
		}
	} else if k8sContainerState.Terminated != nil {
		state = &types.ContainerState{
			Type:        types.TerminatedState,
			ContainerID: k8sContainerState.Terminated.ContainerID,
			ExitCode:    k8sContainerState.Terminated.ExitCode,
			Reason:      k8sContainerState.Terminated.Reason,
			StartedAt:   resource.ISOTime(k8sContainerState.Terminated.StartedAt.Time),
			FinishedAt:  resource.ISOTime(k8sContainerState.Terminated.FinishedAt.Time),
		}
	}

	return state
}

func genCronJobSelector(cli client.Client, namespace, cronjobName string) (labels.Selector, error) {
	k8sCronJob, err := getCronJob(cli, namespace, cronjobName)
	if err != nil {
		return nil, err
	}

	if len(k8sCronJob.Status.Active) == 0 {
		return nil, nil
	}

	var jobUIDs []string
	for _, ref := range k8sCronJob.Status.Active {
		jobUIDs = append(jobUIDs, string(ref.UID))
	}

	requirement, err := labels.NewRequirement("controller-uid", selection.In, jobUIDs)
	if err != nil {
		return nil, err
	}

	return labels.Everything().Add(*requirement), nil
}

func genJobSelector(cli client.Client, namespace, jobName string) (labels.Selector, error) {
	k8sJob, err := getJob(cli, namespace, jobName)
	if err != nil {
		return nil, err
	}

	requirement, err := labels.NewRequirement("controller-uid", selection.Equals, []string{string(k8sJob.UID)})
	if err != nil {
		return nil, err
	}

	return labels.Everything().Add(*requirement), nil
}
