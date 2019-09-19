package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type InnerServiceManager struct {
	clusters *ClusterManager
}

func newInnerServiceManager(clusters *ClusterManager) *InnerServiceManager {
	return &InnerServiceManager{
		clusters: clusters,
	}
}

func (m *InnerServiceManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	resp, err := getInnerServices(cluster.Name, m.clusters.Agent, namespace)
	if err != nil {
		log.Warnf("get innerservices info failed:%s", err.Error())
		return nil
	}
	return resp
}

func getInnerServices(cluster string, agent *clusteragent.AgentManager, namespace string) ([]*types.InnerService, error) {
	url := "/apis/agent.zcloud.cn/v1/namespaces/" + namespace + "/innerservices"
	innerservices := make([]*types.InnerService, 0)
	res := make([]types.InnerService, 0)
	if err := agent.ListResource(cluster, url, &res); err != nil {
		return innerservices, err
	}
	for _, innerservice := range res {
		innerservices = append(innerservices, &innerservice)
	}
	return innerservices, nil
}
