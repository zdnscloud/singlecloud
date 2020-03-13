package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	ca "github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type OuterServiceManager struct {
	clusters *ClusterManager
}

func newOuterServiceManager(clusters *ClusterManager) *OuterServiceManager {
	return &OuterServiceManager{
		clusters: clusters,
	}
}

func (m *OuterServiceManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	var svcs []*types.OuterService
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &svcs); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("list outerservices failed:%s", err.Error()))
	}
	return svcs, nil
}
