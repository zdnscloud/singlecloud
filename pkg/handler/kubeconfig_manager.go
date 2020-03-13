package handler

import (
	"fmt"

	"github.com/zdnscloud/singlecloud/pkg/types"

	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/zke/core/pki"
)

type KubeConfigManager struct {
	clusters *ClusterManager
}

func newKubeConfigManager(clusters *ClusterManager) *KubeConfigManager {
	return &KubeConfigManager{clusters: clusters}
}

func (m *KubeConfigManager) Get(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if !isAdmin(getCurrentUser(ctx)) {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can get cluster kubeconfig")
	}

	id := ctx.Resource.GetID()
	clusterID := ctx.Resource.GetParent().GetID()
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s doesn't exist", clusterID))
	}

	kubeConfig, err := cluster.GetKubeConfig(id, m.clusters.zkeManager.GetDBTable())
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get cluster %s kubeconfig failed %s", clusterID, err.Error()))
	}
	k := &types.KubeConfig{
		User:       id,
		KubeConfig: kubeConfig,
	}
	k.SetID(id)
	k.SetCreationTimestamp(cluster.GetCreationTimestamp())
	k.SetDeletionTimestamp(cluster.GetDeletionTimestamp())
	return k, nil
}

func (m *KubeConfigManager) List(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	if !isAdmin(getCurrentUser(ctx)) {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can get cluster kubeconfig")
	}

	id := ctx.Resource.GetParent().GetID()
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s doesn't exist", id))
	}

	kubeConfig, err := cluster.GetKubeConfig(pki.KubeAdminCertName, m.clusters.zkeManager.GetDBTable())
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get cluster %s kubeconfigs failed %s", id, err.Error()))
	}
	k := &types.KubeConfig{
		User:       pki.KubeAdminCertName,
		KubeConfig: kubeConfig,
	}
	k.SetID(k.User)
	k.SetCreationTimestamp(cluster.GetCreationTimestamp())
	k.SetDeletionTimestamp(cluster.GetDeletionTimestamp())
	return []*types.KubeConfig{k}, nil
}
