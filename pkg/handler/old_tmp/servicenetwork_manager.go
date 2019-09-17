package handler

import (
	"encoding/json"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest"
	resttypes "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"reflect"
)

type ServiceNetworkManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newServiceNetworkManager(clusters *ClusterManager) *ServiceNetworkManager {
	return &ServiceNetworkManager{
		clusters: clusters,
	}
}

func (m *ServiceNetworkManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
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
func getServiceNetworks(cluster string, agent *clusteragent.AgentManager) ([]types.ServiceNetwork, error) {
	nets := make([]types.ServiceNetwork, 0)
	url := "/apis/agent.zcloud.cn/v1/servicenetworks"
	res, err := agent.GetData(cluster, url)
	if err != nil {
		return nets, err
	}
	s := reflect.ValueOf(res)
	for i := 0; i < s.Len(); i++ {
		newp := new(types.ServiceNetwork)
		p := s.Index(i).Interface()
		tmp, _ := json.Marshal(&p)
		json.Unmarshal(tmp, newp)
		nets = append(nets, *newp)
	}
	return nets, nil
}
