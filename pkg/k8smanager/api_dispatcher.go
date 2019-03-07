package k8smanager

import (
	"fmt"
	"net/http"

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
	typ := obj.GetType()
	if typ == types.ClusterType {
		return h.clusterManager.Create(obj.(*types.Cluster), yamlConf)
	}

	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	cluster, found := h.clusterManager.Get(clusterID)
	if found == false {
		return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s doesn't exist", cluster.Name))
	}

	switch typ {
	case types.NamespaceType:
		return newNamespaceManager(cluster).Create(obj.(*types.Namespace), yamlConf)
	case types.DeploymentType:
		return newDeploymentManager(cluster).Create(ancestors[1].GetID(), obj.(*types.Deployment), yamlConf)
	default:
		return nil, nil
	}
}

func (h *Handler) Delete(obj resttypes.Object) *resttypes.APIError {
	return nil
}

func (h *Handler) Update(obj resttypes.Object) (interface{}, *resttypes.APIError) {
	return obj, nil
}

func (h *Handler) List(obj resttypes.Object) interface{} {
	typ := obj.GetType()
	if typ == types.ClusterType {
		return h.clusterManager.List()
	}

	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	cluster, found := h.clusterManager.Get(clusterID)
	if found == false {
		logger.Warn("search for unknown cluster %s", clusterID)
		return nil
	}

	switch typ {
	case types.NodeType:
		return newNodeManager(cluster).List()
	case types.NamespaceType:
		return newNamespaceManager(cluster).List()
	case types.DeploymentType:
		return newDeploymentManager(cluster).List(ancestors[1].GetID())
	default:
		logger.Warn("search for unknown type", obj.GetType())
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

func (h *Handler) OpenConsole(id string, r *http.Request, w http.ResponseWriter) {
	h.clusterManager.OpenConsole(id, r, w)
}
