package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	ca "github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type InnerServiceManager struct {
	clusters *ClusterManager
}

func newInnerServiceManager(clusters *ClusterManager) *InnerServiceManager {
	return &InnerServiceManager{
		clusters: clusters,
	}
}

func (m *InnerServiceManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	var svcs []*types.InnerService
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &svcs); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get innerservices failed:%s", err.Error()))
	}

	return svcs, nil
}
