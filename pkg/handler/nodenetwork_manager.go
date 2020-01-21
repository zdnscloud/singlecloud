package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
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

	resp, err := getNodeNetworks(cluster.Name, clusteragent.GetAgent())
	if err != nil {
		log.Warnf("get nodenetworks info failed:%s", err.Error())
		return nil
	}
	return resp
}

func getNodeNetworks(cluster string, agent *clusteragent.AgentManager) ([]*types.NodeNetwork, error) {
	url := "/apis/agent.zcloud.cn/v1/nodenetworks"
	res := make([]types.NodeNetwork, 0)
	if err := agent.ListResource(cluster, url, &res); err != nil {
		return []*types.NodeNetwork{}, err
	}
	nodeNetworks := make([]*types.NodeNetwork, len(res))
	for i := 0; i < len(res); i++ {
		nodeNetworks[i] = &res[i]
	}
	return nodeNetworks, nil
}
