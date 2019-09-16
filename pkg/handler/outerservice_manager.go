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

type OuterServiceManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newOuterServiceManager(clusters *ClusterManager) *OuterServiceManager {
	return &OuterServiceManager{
		clusters: clusters,
	}
}

func (m *OuterServiceManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	namespace := ctx.Object.GetParent().GetID()
	if cluster == nil {
		return nil
	}

	resp, err := getOuterServices(cluster.Name, m.clusters.Agent, namespace)
	if err != nil {
		log.Warnf("get innerservices info failed:%s", err.Error())
		return nil
	}
	return resp
}

func getOuterServices(cluster string, agent *clusteragent.AgentManager, namespace string) ([]types.OuterService, error) {
	nets := make([]types.OuterService, 0)
	url := "/apis/agent.zcloud.cn/v1/namespaces/" + namespace + "/outerservices"
	res, err := agent.GetData(cluster, url)
	if err != nil {
		return nets, err
	}
	s := reflect.ValueOf(res)
	for i := 0; i < s.Len(); i++ {
		newp := new(types.OuterService)
		p := s.Index(i).Interface()
		tmp, _ := json.Marshal(&p)
		json.Unmarshal(tmp, newp)
		nets = append(nets, *newp)
	}
	return nets, nil
}
