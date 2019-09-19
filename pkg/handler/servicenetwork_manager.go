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

	resp, err := getServiceNetworks(cluster.Name, m.clusters.Agent)
	if err != nil {
		log.Warnf("get servicenetworks info failed:%s", err.Error())
		return nil
	}
	return resp
}
func getServiceNetworks(cluster string, agent *clusteragent.AgentManager) ([]*types.ServiceNetwork, error) {
	svcNetworks := make([]*types.ServiceNetwork, 0)
	url := "/apis/agent.zcloud.cn/v1/servicenetworks"
	res := make([]types.ServiceNetwork, 0)
	if err := agent.ListResource(cluster, url, &res); err != nil {
		return svcNetworks, err
	}
	for _, svcNetwork := range res {
		svcNetworks = append(svcNetworks, &svcNetwork)
	}
	return svcNetworks, nil
}
