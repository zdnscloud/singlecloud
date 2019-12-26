package handler

import (
	"path"
	"strings"

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

func (m *SvcMeshWorkloadManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	var workloads types.SvcMeshWorkloads
	if err := m.clusters.Agent.ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name),
		&workloads); err != nil {
		log.Warnf("list svcmeshworkloads info failed:%s", err.Error())
		return nil
	}

	return workloads
}

func (m *SvcMeshWorkloadManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	workload := &types.SvcMeshWorkload{}
	if err := m.clusters.Agent.GetResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name),
		workload); err != nil {
		log.Warnf("get svcmeshworkload failed:%s", err.Error())
		return nil
	}

	setWorkloadRelativeResourceLink(ctx.Request.URL.Path, workload)
	return workload
}

func genClusterAgentURL(path, cluster string) string {
	urls := strings.SplitAfterN(path, "/clusters/"+cluster, 2)
	return urls[1]
}

func setWorkloadRelativeResourceLink(reqPath string, workload *types.SvcMeshWorkload) {
	workloadLinkPrefix := strings.TrimSuffix(reqPath, workload.GetID())
	for i, in := range workload.Inbound {
		workload.Inbound[i].Link = path.Join(workloadLinkPrefix, in.ID)
	}

	for i, out := range workload.Outbound {
		workload.Outbound[i].Link = path.Join(workloadLinkPrefix, out.ID)
	}

	for i, pod := range workload.Pods {
		workload.Pods[i].Stat.Link = path.Join(reqPath, "svcmeshpods", pod.Stat.ID)
	}
}
