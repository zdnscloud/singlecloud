package handler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type PodManager struct {
	DefaultHandler
	clusters *ClusterManager
}

func newPodManager(clusters *ClusterManager) *PodManager {
	return &PodManager{clusters: clusters}
}

func (m *PodManager) List(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	deploy := obj.GetParent().GetID()
	namespace := obj.GetParent().GetParent().GetID()
	k8sPods, err := getPods(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("list pods info failed:%s", err.Error())
		}
		return nil
	}

	var pods []*types.Pod
	for _, k8sPod := range k8sPods.Items {
		if isPodBelongToDeployment(&k8sPod, deploy) {
			pods = append(pods, k8sPodToSCPod(&k8sPod))
		}
	}
	return pods
}

func (m *PodManager) Get(obj resttypes.Object) interface{} {
	cluster := m.clusters.GetClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	deploy := obj.GetParent().GetID()
	namespace := obj.GetParent().GetParent().GetID()
	pod := obj.(*types.Pod)
	k8sPod, err := getPod(cluster.KubeClient, namespace, pod.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get pod info failed:%s", err.Error())
		}
		return nil
	}

	if isPodBelongToDeployment(k8sPod, deploy) == false {
		logger.Warn("get pod info failed: pod %s not belong to deployment %s", pod.GetID(), deploy)
		return nil
	}

	return k8sPodToSCPod(k8sPod)
}

func getPod(cli client.Client, namespace, name string) (*corev1.Pod, error) {
	pod := corev1.Pod{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &pod)
	return &pod, err
}

func getPods(cli client.Client, namespace string) (*corev1.PodList, error) {
	pods := corev1.PodList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &pods)
	return &pods, err
}

func isPodBelongToDeployment(k8sPod *corev1.Pod, deploy string) bool {
	if k8sPod != nil && k8sPod.ObjectMeta.Labels != nil {
		return k8sPod.ObjectMeta.Labels["app"] == deploy
	}

	return false
}

func k8sPodToSCPod(k8sPod *corev1.Pod) *types.Pod {
	containers := K8sContainersToScContainers(k8sPod.Spec.Containers, k8sPod.Spec.Volumes)

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

	advancedOpts := types.PodAdvancedOptions{
		HostIP:            k8sPod.Status.HostIP,
		PodIP:             k8sPod.Status.PodIP,
		PodConditions:     conditions,
		ContainerStatuses: statuses,
	}

	pod := &types.Pod{
		Name:            k8sPod.Name,
		NodeName:        k8sPod.Spec.NodeName,
		Containers:      containers,
		AdvancedOptions: advancedOpts,
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
