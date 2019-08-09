package handler

import (
	"encoding/json"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"reflect"
)

type NodeNetworkManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newNodeNetworkManager(clusters *ClusterManager) *NodeNetworkManager {
	return &NodeNetworkManager{
		clusters: clusters,
	}
}

func (m *NodeNetworkManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	resp, err := getNodeNetworks(cluster.Name, m.clusters.Agent)
	if err != nil {
		log.Warnf("get podnetworks info failed:%s", err.Error())
		return nil
	}
	return resp
}

func getNodeNetworks(cluster string, agent *clusteragent.AgentManager) ([]types.NodeNetwork, error) {
	nets := make([]types.NodeNetwork, 0)
	url := "/apis/agent.zcloud.cn/v1/nodenetworks"
	res, err := agent.GetData(cluster, url)
	if err != nil {
		return nets, err
	}
	s := reflect.ValueOf(res)
	for i := 0; i < s.Len(); i++ {
		newp := new(types.NodeNetwork)
		p := s.Index(i).Interface()
		tmp, _ := json.Marshal(&p)
		json.Unmarshal(tmp, newp)
		nets = append(nets, *newp)
	}
	return nets, nil
}
