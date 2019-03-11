package k8smanager

import (
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

	cluster := h.getClusterForSubResource(obj)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	switch typ {
	case types.NamespaceType:
		return newNamespaceManager(cluster).Create(obj.(*types.Namespace), yamlConf)
	case types.DeploymentType:
		return newDeploymentManager(cluster).Create(obj.GetParent().GetID(), obj.(*types.Deployment), yamlConf)
	case types.ConfigMapType:
		return newConfigMapManager(cluster).Create(obj.GetParent().GetID(), obj.(*types.ConfigMap), yamlConf)
	default:
		return nil, nil
	}
}

func (h *Handler) Delete(obj resttypes.Object) *resttypes.APIError {
	typ := obj.GetType()
	if typ == types.ClusterType {
		return resttypes.NewAPIError(resttypes.MethodNotAllowed, "delete cluster isn't allowed")
	}

	cluster := h.getClusterForSubResource(obj)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	switch typ {
	case types.NodeType:
		return resttypes.NewAPIError(resttypes.MethodNotAllowed, "delete node isn't allowed")
	case types.NamespaceType:
		return newNamespaceManager(cluster).Delete(obj.(*types.Namespace))
	case types.DeploymentType:
		return newDeploymentManager(cluster).Delete(obj.GetParent().GetID(), obj.(*types.Deployment))
	case types.ConfigMapType:
		return newConfigMapManager(cluster).Delete(obj.GetParent().GetID(), obj.(*types.ConfigMap))
	default:
		logger.Warn("search for unknown type", obj.GetType())
		return nil
	}

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

	cluster := h.getClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	switch typ {
	case types.NodeType:
		return newNodeManager(cluster).List()
	case types.NamespaceType:
		return newNamespaceManager(cluster).List()
	case types.DeploymentType:
		return newDeploymentManager(cluster).List(obj.GetParent().GetID())
	case types.ConfigMapType:
		return newConfigMapManager(cluster).List(obj.GetParent().GetID())
	default:
		logger.Warn("search for unknown type", obj.GetType())
		return nil
	}
}

func (h *Handler) Get(obj resttypes.Object) interface{} {
	typ := obj.GetType()
	if typ == types.ClusterType {
		return h.clusterManager.Get(obj.GetID())
	}

	cluster := h.getClusterForSubResource(obj)
	if cluster == nil {
		return nil
	}

	switch typ {
	case types.NodeType:
		return newNodeManager(cluster).Get(obj.(*types.Node))
	case types.NamespaceType:
		return newNamespaceManager(cluster).Get(obj.(*types.Namespace))
	case types.DeploymentType:
		return newDeploymentManager(cluster).Get(obj.GetParent().GetID(), obj.(*types.Deployment))
	case types.ConfigMapType:
		return newConfigMapManager(cluster).Get(obj.GetParent().GetID(), obj.(*types.ConfigMap))
	default:
		logger.Warn("search for unknown type", obj.GetType())
		return nil
	}
}

func (h *Handler) Action(obj resttypes.Object, action string, params map[string]interface{}) (interface{}, *resttypes.APIError) {
	return params, nil
}

func (h *Handler) OpenConsole(id string, r *http.Request, w http.ResponseWriter) {
	h.clusterManager.OpenConsole(id, r, w)
}

func (h *Handler) getClusterForSubResource(obj resttypes.Object) *types.Cluster {
	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	cluster := h.clusterManager.Get(clusterID)
	if cluster == nil {
		logger.Warn("search for unknown cluster %s", clusterID)
	}
	return cluster
}
