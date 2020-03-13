package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	ca "github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type MetricManager struct {
	clusters *ClusterManager
}

func newMetricManager(clusters *ClusterManager) *MetricManager {
	return &MetricManager{clusters: clusters}
}

func (m *MetricManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	var metrics []*types.Metric
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &metrics); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list metrics failed:%s", err.Error()))
	}

	return metrics, nil
}
