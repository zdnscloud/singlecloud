package authz

import (
	"context"

	"github.com/zdnscloud/zke/pkg/k8s"
	"github.com/zdnscloud/zke/pkg/log"
	"k8s.io/client-go/kubernetes"
)

func ApplyDefaultPodSecurityPolicy(ctx context.Context, k8sClient *kubernetes.Clientset) error {
	log.Infof(ctx, "[authz] Applying default PodSecurityPolicy")

	if err := k8s.UpdatePodSecurityPolicyFromYaml(k8sClient, defaultPodSecurityPolicy); err != nil {
		return err
	}
	log.Infof(ctx, "[authz] Default PodSecurityPolicy applied successfully")
	return nil
}

func ApplyDefaultPodSecurityPolicyRole(ctx context.Context, k8sClient *kubernetes.Clientset) error {
	log.Infof(ctx, "[authz] Applying default PodSecurityPolicy Role and RoleBinding")

	if err := k8s.UpdateRoleFromYaml(k8sClient, defaultPodSecurityRole); err != nil {
		return err
	}
	if err := k8s.UpdateRoleBindingFromYaml(k8sClient, defaultPodSecurityRoleBinding); err != nil {
		return err
	}
	log.Infof(ctx, "[authz] Default PodSecurityPolicy Role and RoleBinding applied successfully")
	return nil
}
