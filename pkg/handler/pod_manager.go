package handler

import (
	"context"
	"fmt"

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

type PodManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newPodManager(clusters *ClusterManager) *PodManager {
	return &PodManager{clusters: clusters}
}

func (m *PodManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	deploy := ctx.Object.GetParent().GetID()
	namespace := ctx.Object.GetParent().GetParent().GetID()
	k8sDeploy, err := getDeployment(cluster.KubeClient, namespace, deploy)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get deployment info failed:%s", err.Error())
		}
		return nil
	}

	if k8sDeploy.Spec.Selector == nil {
		logger.Warn("deployment %s has no selector", deploy)
		return nil
	}

	k8sPods, err := getPods(cluster.KubeClient, namespace, k8sDeploy.Spec.Selector)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("list pods info failed:%s", err.Error())
		}
		return nil
	}

	var pods []*types.Pod
	for _, k8sPod := range k8sPods.Items {
		pods = append(pods, k8sPodToSCPod(&k8sPod))
	}
	return pods
}

func (m *PodManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetParent().GetID()
	pod := ctx.Object.(*types.Pod)
	k8sPod, err := getPod(cluster.KubeClient, namespace, pod.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get pod info failed:%s", err.Error())
		}
		return nil
	}

	return k8sPodToSCPod(k8sPod)
}

func (m *PodManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetParent().GetID()
	pod := ctx.Object.(*types.Pod)
	err := deletePod(cluster.KubeClient, namespace, pod.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("pod %s desn't exist", pod.GetID()))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete pod failed %s", err.Error()))
		}
	}
	return nil
}

func getPod(cli client.Client, namespace, name string) (*corev1.Pod, error) {
	pod := corev1.Pod{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &pod)
	return &pod, err
}

func getPods(cli client.Client, namespace string, selector *metav1.LabelSelector) (*corev1.PodList, error) {
	pods := corev1.PodList{}
	opts := &client.ListOptions{Namespace: namespace}
	labels, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}

	opts.LabelSelector = labels
	err = cli.List(context.TODO(), opts, &pods)
	return &pods, err
}

func deletePod(cli client.Client, namespace, name string) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), pod)
}

func k8sPodToSCPod(k8sPod *corev1.Pod) *types.Pod {
	containers := k8sContainersToScContainers(k8sPod.Spec.Containers, k8sPod.Spec.Volumes)

	var conditions []types.PodCondition
	for _, condition := range k8sPod.Status.Conditions {
		conditions = append(conditions, types.PodCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastProbeTime:      resttypes.ISOTime(condition.LastProbeTime.Time),
			LastTransitionTime: resttypes.ISOTime(condition.LastTransitionTime.Time),
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
		Containers: containers,
		Status:     podStatus,
	}
	pod.SetID(k8sPod.Name)
	pod.SetType(types.PodType)
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
			StartedAt: resttypes.ISOTime(k8sContainerState.Running.StartedAt.Time),
		}
	} else if k8sContainerState.Terminated != nil {
		state = &types.ContainerState{
			Type:        types.TerminatedState,
			ContainerID: k8sContainerState.Terminated.ContainerID,
			ExitCode:    k8sContainerState.Terminated.ExitCode,
			Reason:      k8sContainerState.Terminated.Reason,
			StartedAt:   resttypes.ISOTime(k8sContainerState.Terminated.StartedAt.Time),
			FinishedAt:  resttypes.ISOTime(k8sContainerState.Terminated.FinishedAt.Time),
		}
	}

	return state
}
