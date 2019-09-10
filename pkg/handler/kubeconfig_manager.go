package handler

import (
	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/zke/core/pki"
)

type KubeConfigManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newKubeConfigManager(clusters *ClusterManager) *KubeConfigManager {
	return &KubeConfigManager{clusters: clusters}
}

func (m *KubeConfigManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	inner := ctx.Object.(*types.KubeConfig)

	kubeConfig, err := cluster.GetKubeConfig(inner.User, m.clusters.GetDB())
	if err != nil {
		return nil
	}
	inner.KubeConfig = kubeConfig
	inner.SetID(inner.User)
	return inner
}

func (m *KubeConfigManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	kubeConfig, err := cluster.GetKubeConfig(pki.KubeAdminCertName, m.clusters.GetDB())
	if err != nil {
		return nil
	}
	kubeConfigs := []*types.KubeConfig{&types.KubeConfig{
		User:       pki.KubeAdminCertName,
		KubeConfig: kubeConfig,
	}}
	return kubeConfigs
}
