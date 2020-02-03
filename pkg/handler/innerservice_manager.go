package handler

import (
	"github.com/zdnscloud/cement/log"
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

func (m *InnerServiceManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	var svcs []*types.InnerService
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &svcs); err != nil {
		log.Warnf("get innerservices info failed:%s", err.Error())
		return nil
	}

	return svcs
}
