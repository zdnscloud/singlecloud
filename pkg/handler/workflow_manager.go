package handler

import (
	"context"
	"fmt"

	// "github.com/zdnscloud/cement/log"
	// "github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	// apierrors "k8s.io/apimachinery/pkg/api/errors"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/zdnscloud/cement/randomdata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	zcloudWorkFlowLabelKey               = "workflow.zdns.cn"
	zcloudWorkFlowClusterRoleBindingName = "zcloud-workflow-deployer"
)

type WorkFlowManager struct {
	clusters *ClusterManager
}

func newWorkFlowManager(clusters *ClusterManager) *WorkFlowManager {
	return &WorkFlowManager{
		clusters: clusters,
	}
}

func (m *WorkFlowManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	return nil, nil
}

func createSecretsForWorkFlow(client client.Client, namespace string, wf *types.WorkFlow) error {
	if wf.Git.User != "" && wf.Git.Password != "" {
		if err := client.Create(context.TODO(), getBasicAuthSecret(genWFResourceName(wf.Name), namespace, wf.Name, wf.Git.User, wf.Git.Password)); err != nil {
			return err
		}
	}

	return client.Create(context.TODO(), getBasicAuthSecret(genWFResourceName(wf.Name), namespace, wf.Name, wf.Image.RegistryUser, wf.Image.RegistryPassword))
}

func genWFResourceName(workFlowName string) string {
	return fmt.Sprintf("%s-%s", workFlowName, randomdata.RandString(12))
}

func getBasicAuthSecret(name, namespace, labelValue, user, password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				zcloudWorkFlowLabelKey: labelValue,
			},
		},
		Type: corev1.SecretTypeBasicAuth,
		StringData: map[string]string{
			"username": user,
			"password": password,
		},
	}
}

func createServiceAccountForWorkFlow(client client.Client, name, namespace string, secrets ...string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace},
		Secrets: []corev1.ObjectReference{},
	}

	for _, secret := range secrets {
		sa.Secrets = append(sa.Secrets, corev1.ObjectReference{Name: secret})
	}
	return client.Create(context.TODO(), sa)
}

func addSaToClusterRoleBinding(client client.Client, saName, saNamespace string) error {
	crb := &rbacv1.ClusterRoleBinding{}
	if err := client.Get(context.TODO(), k8stypes.NamespacedName{Name: zcloudWorkFlowClusterRoleBindingName}, crb); err != nil {
		return err
	}
	crb.Subjects = append(crb.Subjects, rbacv1.Subject{
		Kind:      rbacv1.ServiceAccountKind,
		Name:      saName,
		Namespace: saNamespace,
	})
	return client.Update(context.TODO(), crb)
}

func removeSaFromClusterRoleBinding(client client.Client, saName, saNamespace string) error {
	crb := &rbacv1.ClusterRoleBinding{}
	if err := client.Get(context.TODO(), k8stypes.NamespacedName{Name: zcloudWorkFlowClusterRoleBindingName}, crb); err != nil {
		return err
	}

	subjects := []rbacv1.Subject{}
	for _, subject := range crb.Subjects {
		if subject.Name == saName && subject.Namespace == saNamespace {
			continue
		}
		subjects = append(subjects, subject)
	}
	crb.Subjects = subjects
	return client.Update(context.TODO(), crb)
}

func createPipelineResource(client client.Client, namespace string, wf *types.WorkFlow) error {
	r := &tektonv1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wf.Name,
			Namespace: namespace},
		Spec: tektonv1.PipelineResourceSpec{
			Type: tektonv1.PipelineResourceTypeGit,
			Params: []tektonv1.ResourceParam{
				tektonv1.ResourceParam{
					Name:  "url",
					Value: wf.Git.RepositoryURL,
				},
				tektonv1.ResourceParam{
					Name:  "revision",
					Value: wf.Git.Revision,
				},
			},
		},
	}
	return client.Create(context.TODO(), r)
}

func (m *WorkFlowManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	return nil
}

func (m *WorkFlowManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}
	return nil
}

func (m *WorkFlowManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, nil
	}
	return nil, nil
}

func (m *WorkFlowManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}
	return nil
}
