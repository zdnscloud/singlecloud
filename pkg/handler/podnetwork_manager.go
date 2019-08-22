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

type PodNetworkManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newPodNetworkManager(clusters *ClusterManager) *PodNetworkManager {
	return &PodNetworkManager{
		clusters: clusters,
	}
}

func (m *PodNetworkManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
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

func getPodNetworks(cluster string, agent *clusteragent.AgentManager) ([]types.PodNetwork, error) {
	nets := make([]types.PodNetwork, 0)
	url := "/apis/agent.zcloud.cn/v1/podnetworks"
	res, err := agent.GetData(cluster, url)
	if err != nil {
		return nets, err
	}
	s := reflect.ValueOf(res)
	for i := 0; i < s.Len(); i++ {
		newp := new(types.PodNetwork)
		p := s.Index(i).Interface()
		tmp, _ := json.Marshal(&p)
		json.Unmarshal(tmp, newp)
		nets = append(nets, *newp)
	}
	return nets, nil
}
