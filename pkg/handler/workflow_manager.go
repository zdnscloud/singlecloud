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
	// apierrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/zdnscloud/cement/randomdata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	zcloudWorkFlowLabelKey = "workflow.zdns.cn"
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
