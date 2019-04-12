package handler

import (
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type ServiceLinkManager struct {
	DefaultHandler
	clusters *ClusterManager
}

func newServiceLinkManager(clusters *ClusterManager) *ServiceLinkManager {
	return &ServiceLinkManager{clusters: clusters}
}

func (m *ServiceLinkManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	switch ctx.Object.GetType() {
	case types.InnerServiceType:
		return cluster.ServiceCache.GetInnerServices(namespace)
	case types.OuterServiceType:
		return cluster.ServiceCache.GetOuterServices(namespace)
	}
	return nil
}
