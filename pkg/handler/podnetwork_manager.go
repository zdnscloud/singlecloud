package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
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

	resp, err := getPodNetworks(cluster.Name, m.clusters.Agent)
	if err != nil {
		log.Warnf("get podnetworks info failed:%s", err.Error())
		return nil
	}
	return resp
}

func getPodNetworks(cluster string, agent *clusteragent.AgentManager) ([]*types.PodNetwork, error) {
	podNetworks := make([]*types.PodNetwork, 0)
	url := "/apis/agent.zcloud.cn/v1/podnetworks"
	res := make([]types.PodNetwork, 0)
	if err := agent.ListResource(cluster, url, &res); err != nil {
		return podNetworks, err
	}
	for _, podNetwork := range res {
		podNetworks = append(podNetworks, &podNetwork)
	}
	return podNetworks, nil
}
