package handler

import (
	"fmt"
	"path"
	"strings"

	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	ca "github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type SvcMeshWorkloadManager struct {
	clusters *ClusterManager
}

func newSvcMeshWorkloadManager(clusters *ClusterManager) *SvcMeshWorkloadManager {
	return &SvcMeshWorkloadManager{clusters: clusters}
}

func (m *SvcMeshWorkloadManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	var workloads types.SvcMeshWorkloads
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &workloads); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("list svcmeshworkloads failed:%s", err.Error()))
	}

	return workloads, nil
}

func (m *SvcMeshWorkloadManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	workload := &types.SvcMeshWorkload{}
	if err := ca.GetAgent().GetResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), workload); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("get svcmeshworkload %s failed:%s", workload.GetID(), err.Error()))
	}

	setWorkloadRelativeResourceLink(ctx.Request.URL.Path, workload)
	return workload, nil
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
