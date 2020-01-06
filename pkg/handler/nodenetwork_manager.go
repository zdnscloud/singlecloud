package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type NodeNetworkManager struct {
	clusters *ClusterManager
}

func newNodeNetworkManager(clusters *ClusterManager) *NodeNetworkManager {
	return &NodeNetworkManager{
		clusters: clusters,
	}
}

func (m *NodeNetworkManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	var networks []*types.NodeNetwork
	if err := m.clusters.Agent.ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &networks); err != nil {
		log.Warnf("get nodenetworks info failed:%s", err.Error())
		return nil
	}
	return networks
}
