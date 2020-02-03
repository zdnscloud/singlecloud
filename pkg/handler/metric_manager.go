package handler

import (
	"github.com/zdnscloud/cement/log"
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

func (m *MetricManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	var metrics []*types.Metric
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &metrics); err != nil {
		log.Warnf("list metrics info failed: %s", err.Error())
		return nil
	}

	return metrics
}
