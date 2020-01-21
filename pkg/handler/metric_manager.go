package handler

import (
	"strings"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
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

	url := "/apis/agent.zcloud.cn/v1" + strings.SplitAfterN(ctx.Request.URL.Path, "/clusters/"+cluster.Name, 2)[1]
	var metrics []*types.Metric
	if err := clusteragent.GetAgent().ListResource(cluster.Name, url, &metrics); err != nil {
		log.Warnf("list metrics info failed: %s", err.Error())
		return nil
	}

	return metrics
}
