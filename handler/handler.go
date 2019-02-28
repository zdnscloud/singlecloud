package handler

import (
	"github.com/zdnscloud/cement/uuid"
	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/logger"
	"github.com/zdnscloud/singlecloud/types/cluster"
	"github.com/zdnscloud/singlecloud/types/node"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Create(obj types.Object) (interface{}, error) {
	if obj.GetParent().Name != "" {
		logger.GetLogger().Debug("create %s in %s %s",
			obj.GetType(), obj.GetParent().Name, obj.GetParent().ID)
	} else {
		logger.GetLogger().Debug("create %s", obj.GetType())
	}

	return getTestCluster(), nil
}

func (h *Handler) Delete(obj types.Object) error {
	if obj.GetParent().Name != "" {
		logger.GetLogger().Debug("delete %s %s from %s %s",
			obj.GetType(), obj.GetID(), obj.GetParent().Name, obj.GetParent().ID)
	} else {
		logger.GetLogger().Debug("delete %s %s", obj.GetType(), obj.GetID())
	}
	return nil
}

func (h *Handler) BatchDelete(obj types.Object) error {
	if obj.GetParent().Name != "" {
		logger.GetLogger().Debug("delete all %s from %s %s",
			obj.GetType(), obj.GetParent().Name, obj.GetParent().ID)
	} else {
		logger.GetLogger().Debug("delete all %s", obj.GetType())
	}
	return nil
}

func (h *Handler) Update(objType types.ObjectType, objId types.ObjectID, obj types.Object) (interface{}, error) {
	if obj.GetParent().Name != "" {
		logger.GetLogger().Debug("update %s %s in %s %s",
			objType.GetType(), objId.GetID(), obj.GetParent().Name, obj.GetParent().ID)
	} else {
		logger.GetLogger().Debug("update %s %s", objType.GetType(), objId.GetID())
	}

	return obj, nil
}

func (h *Handler) List(obj types.Object) interface{} {
	var result interface{}
	if obj.GetParent().Name != "" {
		logger.GetLogger().Debug("get all %s in %s %s",
			obj.GetType(), obj.GetParent().Name, obj.GetParent().ID)
		result = []node.Node{getTestNode()}
	} else {
		logger.GetLogger().Debug("get all %s", obj.GetType())
		result = []cluster.Cluster{getTestCluster()}
	}
	return result
}

func (h *Handler) Get(obj types.Object) interface{} {
	var result interface{}
	if obj.GetParent().Name != "" {
		logger.GetLogger().Debug("get %s %s in %s %s",
			obj.GetType(), obj.GetID(), obj.GetParent().Name, obj.GetParent().ID)
		result = getTestNode()
	} else {
		logger.GetLogger().Debug("get %s %s", obj.GetType(), obj.GetID())
		result = getTestCluster()
	}
	return result
}

func (h *Handler) Action(obj types.Object, action string, params map[string]interface{}) (interface{}, error) {
	if obj.GetParent().Name != "" {
		logger.GetLogger().Debug("run action %s with params %v for %s %s in %s %s",
			action, params, obj.GetType(), obj.GetID(), obj.GetParent().Name, obj.GetParent().ID)
	} else {
		logger.GetLogger().Debug("run action %s with params %v for %s %s", action, params, obj.GetType(), obj.GetID())
	}
	return params, nil
}

//Just for test
var (
	clusterID, _ = uuid.Gen()
	nodeID, _    = uuid.Gen()
)

func getTestNode() node.Node {
	return node.Node{
		ID:                   nodeID,
		Type:                 "node",
		Name:                 "testNode",
		Address:              "127.0.0.1",
		Role:                 []string{"etcd", "controlplane", "worker"},
		Labels:               map[string]interface{}{"node-role.kubernetes.io/controlplane": "true"},
		Annotations:          map[string]interface{}{"volumes.kubernetes.io/controller-managed-attach-detach": "true"},
		Status:               true,
		OperatingSystem:      "Linux",
		OperatingSystemImage: "Ubuntu 16.04.4 LTS",
		DockerVersion:        "17.3.2",
		Cpu:                  32,
		CpuUsedRatio:         "1.6%",
		Memory:               "128Gi",
		MemoryUsedRatio:      "2.6%",
		CreationTimestamp:    "2019-02-13T05:46:29Z",
	}
}

func getTestCluster() cluster.Cluster {
	return cluster.Cluster{
		ID:                clusterID,
		Type:              "cluster",
		Name:              "testCluster",
		NodesCount:        1,
		Version:           "v1.13.1",
		CreationTimestamp: "2018-12-13T10:31:33Z",
	}
}
