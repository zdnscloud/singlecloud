package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	ca "github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type ServiceNetworkManager struct {
	clusters *ClusterManager
}

func newServiceNetworkManager(clusters *ClusterManager) *ServiceNetworkManager {
	return &ServiceNetworkManager{
		clusters: clusters,
	}
}

func (m *ServiceNetworkManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	var networks []*types.ServiceNetwork
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &networks); err != nil {
		log.Warnf("get servicenetworks info failed:%s", err.Error())
		return nil
	}
	return networks
}
