package handler

import (
	"strings"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type SvcMeshWorkloadGroupManager struct {
	clusters *ClusterManager
}

func newSvcMeshWorkloadGroupManager(clusters *ClusterManager) *SvcMeshWorkloadGroupManager {
	return &SvcMeshWorkloadGroupManager{clusters: clusters}
}

func (m *SvcMeshWorkloadGroupManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	var groups types.SvcMeshWorkloadGroups
	if err := m.clusters.Agent.ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &groups); err != nil {
		log.Warnf("list svcmeshworkloadgroups info failed:%s", err.Error())
		return nil
	}

	return groups
}

func genClusterAgentURL(path, cluster string) string {
	urls := strings.SplitAfterN(path, "/clusters/"+cluster, 2)
	return urls[1]
}
