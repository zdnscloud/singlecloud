package helper

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/cache"
)

//copy code from k8s.io/kubernetes/pkg/printers
func GetPodState(pod *corev1.Pod) string {
	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}

	if !initializing {
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == "NodeLost" {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	return reason
}

func GetPodOwner(c cache.Cache, pod *corev1.Pod) (string, string, error) {
	if len(pod.OwnerReferences) != 1 {
		return "", "", fmt.Errorf("pod no owner refernces")
	}

	owner := pod.OwnerReferences[0]
	if owner.Kind != "ReplicaSet" {
		return strings.ToLower(owner.Kind), owner.Name, nil
	}

	var k8srs appsv1.ReplicaSet
	err := c.Get(context.TODO(), k8stypes.NamespacedName{pod.Namespace, owner.Name}, &k8srs)
	if err != nil {
		return "", "", fmt.Errorf("get replicaset failed:%s", err.Error())
	}

	if len(k8srs.OwnerReferences) != 1 {
		return "", "", fmt.Errorf("replicaset OwnerReferences is strange:%v", k8srs.OwnerReferences)
	}

	owner = k8srs.OwnerReferences[0]
	if owner.Kind != "Deployment" {
		return "", "", fmt.Errorf("replicaset parent is not deployment but %v", owner.Kind)
	}

	return strings.ToLower(owner.Kind), owner.Name, nil
}
