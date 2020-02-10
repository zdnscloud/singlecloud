package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	ca "github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type PodNetworkManager struct {
	clusters *ClusterManager
}

func newPodNetworkManager(clusters *ClusterManager) *PodNetworkManager {
	return &PodNetworkManager{
		clusters: clusters,
	}
}

func (m *PodNetworkManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	var networks []*types.PodNetwork
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &networks); err != nil {
		log.Warnf("get podnetworks info failed:%s", err.Error())
		return nil
	}
	return networks
}
