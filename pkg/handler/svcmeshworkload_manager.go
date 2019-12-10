package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type SvcMeshWorkloadManager struct {
	clusters *ClusterManager
}

func newSvcMeshWorkloadManager(clusters *ClusterManager) *SvcMeshWorkloadManager {
	return &SvcMeshWorkloadManager{clusters: clusters}
}

func (m *SvcMeshWorkloadManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	workload := &types.SvcMeshWorkload{}
	if err := m.clusters.Agent.GetResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), workload); err != nil {
		log.Warnf("get svcmeshworkload failed:%s", err.Error())
		return nil
	}

	return workload
}
