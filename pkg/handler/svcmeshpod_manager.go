package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type SvcMeshPodManager struct {
	clusters *ClusterManager
}

func newSvcMeshPodManager(clusters *ClusterManager) *SvcMeshPodManager {
	return &SvcMeshPodManager{clusters: clusters}
}

func (m *SvcMeshPodManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	pod := &types.SvcMeshPod{}
	if err := m.clusters.Agent.GetResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), pod); err != nil {
		log.Warnf("get svcmeshpod failed:%s", err.Error())
		return nil
	}

	return pod
}
