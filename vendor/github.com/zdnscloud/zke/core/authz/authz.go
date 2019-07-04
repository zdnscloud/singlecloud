package authz

import (
	"context"

	"github.com/zdnscloud/zke/pkg/k8s"
	"github.com/zdnscloud/zke/pkg/log"
	"k8s.io/client-go/kubernetes"
)

func ApplyJobDeployerServiceAccount(ctx context.Context, k8sClient *kubernetes.Clientset) error {
	log.Infof(ctx, "[authz] Creating zke-job-deployer ServiceAccount")

	if err := k8s.UpdateClusterRoleBindingFromYaml(k8sClient, jobDeployerClusterRoleBinding); err != nil {
		return err
	}
	if err := k8s.UpdateServiceAccountFromYaml(k8sClient, jobDeployerServiceAccount); err != nil {
		return err
	}
	log.Infof(ctx, "[authz] zke-job-deployer ServiceAccount created successfully")
	return nil
}

func ApplySystemNodeClusterRoleBinding(ctx context.Context, k8sClient *kubernetes.Clientset) error {
	if err := k8s.UpdateClusterRoleBindingFromYaml(k8sClient, systemNodeClusterRoleBinding); err != nil {
		return err
	}
	log.Infof(ctx, "[authz] system:node ClusterRoleBinding created successfully")
	return nil
}
