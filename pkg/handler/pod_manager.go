package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
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

	namespace := ctx.Object.GetParent().GetParent().GetID()
	selector, err := getPodParentSelector(cluster.KubeClient, namespace,
		ctx.Object.GetParent().GetType(), ctx.Object.GetParent().GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get deployment info failed:%s", err.Error())
		}
		return nil
	}

	if selector == nil {
		log.Warnf("%s %s has no selector", ctx.Object.GetParent().GetType(), ctx.Object.GetParent().GetID())
		return nil
	}

	k8sPods, err := getPods(cluster.KubeClient, namespace, selector)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list pods info failed:%s", err.Error())
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
			log.Warnf("get pod info failed:%s", err.Error())
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

func getPodParentSelector(cli client.Client, namespace string, typ string, name string) (*metav1.LabelSelector, error) {
	switch typ {
	case types.DeploymentType:
		k8sDeploy, err := getDeployment(cli, namespace, name)
		if err != nil {
			return nil, err
		}

		return k8sDeploy.Spec.Selector, nil
	case types.DaemonSetType:
		k8sDaemonSet, err := getDaemonSet(cli, namespace, name)
		if err != nil {
			return nil, err
		}

		return k8sDaemonSet.Spec.Selector, nil
	case types.StatefulSetType:
		k8sStatefulSet, err := getStatefulSet(cli, namespace, name)
		if err != nil {
			return nil, err
		}

		return k8sStatefulSet.Spec.Selector, nil
	default:
		return nil, fmt.Errorf("pod no such parent %v", typ)
	}
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

	//pod state is the first non-running reason
	//or running if all container is in running state
	state := types.RunningState
	for _, status := range statuses {
		if status.State.Type != types.RunningState {
			state = status.State.Reason
			break
		}
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
		State:      state,
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
