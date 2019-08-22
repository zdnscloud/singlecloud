package helper

import (
	corev1 "k8s.io/api/core/v1"
)

func IsNodeReady(node *corev1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady &&
			cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
