package handler

import (
	"github.com/zdnscloud/singlecloud/pkg/types"

	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/zke/core/pki"
)

type KubeConfigManager struct {
	clusters *ClusterManager
}

func newKubeConfigManager(clusters *ClusterManager) *KubeConfigManager {
	return &KubeConfigManager{clusters: clusters}
}

func (m *KubeConfigManager) Get(ctx *restresource.Context) restresource.Resource {
	if !isAdmin(getCurrentUser(ctx)) {
		return nil
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	id := ctx.Resource.GetID()

	kubeConfig, err := cluster.GetKubeConfig(id, m.clusters.zkeManager.GetDBTable())
	if err != nil {
		return nil
	}
	k := &types.KubeConfig{
		User:       id,
		KubeConfig: kubeConfig,
	}
	k.SetID(id)
	k.SetCreationTimestamp(cluster.ToScCluster().GetCreationTimestamp())
	k.SetDeletionTimestamp(cluster.ToScCluster().GetDeletionTimestamp())
	return k
}

func (m *KubeConfigManager) List(ctx *restresource.Context) interface{} {
	if !isAdmin(getCurrentUser(ctx)) {
		return nil
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	kubeConfig, err := cluster.GetKubeConfig(pki.KubeAdminCertName, m.clusters.zkeManager.GetDBTable())
	if err != nil {
		return nil
	}
	k := &types.KubeConfig{
		User:       pki.KubeAdminCertName,
		KubeConfig: kubeConfig,
	}
	k.SetID(k.User)
	k.SetCreationTimestamp(cluster.ToScCluster().GetCreationTimestamp())
	k.SetDeletionTimestamp(cluster.ToScCluster().GetDeletionTimestamp())
	return []*types.KubeConfig{k}
}
