package k8smanager

import (
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type Handler struct {
	clusterManager *ClusterManager
}

func NewHandler() *Handler {
	return &Handler{
		clusterManager: newClusterManager(),
	}
}

func (h *Handler) Create(obj resttypes.Object, yamlConf []byte) (interface{}, *resttypes.APIError) {
	if cluster, ok := obj.(*types.Cluster); ok {
		return h.clusterManager.Create(cluster, yamlConf)
	}
	return nil, nil
}

func (h *Handler) Delete(obj resttypes.Object) *resttypes.APIError {
	return nil
}

func (h *Handler) Update(obj resttypes.Object) (interface{}, *resttypes.APIError) {
	return obj, nil
}

func (h *Handler) List(obj resttypes.Object) interface{} {
	switch obj.GetType() {
	case types.ClusterType:
		return h.clusterManager.List()
	case types.NodeType:
		id := obj.GetParent().ID
		cluster, found := h.clusterManager.Get(id)
		if found == false {
			logger.Warn("search for unknown cluster %s", id)
			return nil
		}

		k8sNodes, err := getNodes(cluster.KubeClient)
		if err != nil {
			logger.Error("get nodes failed %s", err.Error())
			return nil
		}

		var nodes []*types.Node
		for _, k8sNode := range k8sNodes.Items {
			nodes = append(nodes, k8sNodeToSCNode(&k8sNode))
		}
		return nodes

	default:
		return nil
	}
}

func (h *Handler) Get(obj resttypes.Object) interface{} {
	if _, ok := obj.(*types.Cluster); ok {
		c, _ := h.clusterManager.Get(obj.GetID())
		return c
	} else {
		return nil
	}
}

func (h *Handler) Action(obj resttypes.Object, action string, params map[string]interface{}) (interface{}, *resttypes.APIError) {
	return params, nil
}
