package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
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

	resp, err := getServiceNetworks(cluster.Name, clusteragent.GetAgent())
	if err != nil {
		log.Warnf("get servicenetworks info failed:%s", err.Error())
		return nil
	}
	return resp
}
func getServiceNetworks(cluster string, agent *clusteragent.AgentManager) ([]*types.ServiceNetwork, error) {
	url := "/apis/agent.zcloud.cn/v1/servicenetworks"
	res := make([]types.ServiceNetwork, 0)
	if err := agent.ListResource(cluster, url, &res); err != nil {
		return []*types.ServiceNetwork{}, err
	}
	svcNetworks := make([]*types.ServiceNetwork, len(res))
	for i := 0; i < len(res); i++ {
		svcNetworks[i] = &res[i]
	}
	return svcNetworks, nil
}
