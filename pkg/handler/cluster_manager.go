package handler

import (
	"fmt"
	"time"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gok8s/exec"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/event"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	ZCloudNamespace = "zcloud"
	ZCloudAdmin     = "zcloud-cluster-admin"
	ZCloudReadonly  = "zcloud-cluster-readonly"

	MaxEventCount = 4096
)

type ClusterManager struct {
	DefaultHandler
	clusters []*types.Cluster
}

func newClusterManager() *ClusterManager {
	return &ClusterManager{}
}

func (m *ClusterManager) GetClusterForSubResource(obj resttypes.Object) *types.Cluster {
	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	return m.get(clusterID)
}

func (m *ClusterManager) Create(obj resttypes.Object, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := obj.(*types.Cluster)
	if c := m.get(cluster.Name); c != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate cluster name")
	}

	cluster.SetID(cluster.Name)
	k8sconf, err := config.BuildConfig(yamlConf)
	if err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("invalid cluster config:%s", err.Error()))
	}

	cli, err := client.New(k8sconf, client.Options{})
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("connect to cluster failed:%s", err.Error()))
	}

	executor, err := exec.New(k8sconf)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("connect to cluster failed:%s", err.Error()))
	}

	nodes, err := getNodes(cli)
	if err != nil {
		logger.Error("get nodes failed:%s", err.Error())
	}
	cluster.NodesCount = len(nodes.Items)

	version, err := cli.ServerVersion()
	if err != nil {
		logger.Error("get version failed:%s", err.Error())
	} else {
		cluster.Version = version.GitVersion
	}
	cluster.KubeClient = cli
	cluster.Executor = executor

	eventWatcher, err := event.New(k8sconf, MaxEventCount)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("add cluster event watcher:%s", err.Error()))
	}
	cluster.EventWatcher = eventWatcher

	if err := initCluster(cluster); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("init cluster failed:%s", err.Error()))
	}

	cluster.SetCreationTimestamp(time.Now())
	m.clusters = append(m.clusters, cluster)
	return cluster, nil
}

func (m *ClusterManager) Get(obj resttypes.Object) interface{} {
	return m.get(obj.GetID())
}

func (m *ClusterManager) get(id string) *types.Cluster {
	for _, c := range m.clusters {
		if c.GetID() == id {
			return c
		}
	}
	return nil
}

func (m *ClusterManager) List(obj resttypes.Object) interface{} {
	return m.clusters
}

func initCluster(cluster *types.Cluster) error {
	imported, err := isClusterImportBefore(cluster)
	if err != nil {
		return err
	}
	if imported {
		logger.Info("cluster %s has been imported before", cluster.Name)
		return nil
	}

	cli := cluster.KubeClient
	if err := createNamespace(cli, ZCloudNamespace); err != nil {
		return err
	}

	if err := createRole(cluster, ZCloudAdmin, ClusterAdmin); err != nil {
		return err
	}
	if err := createRole(cluster, ZCloudReadonly, ClusterAdmin); err != nil {
		return err
	}
	return nil
}

func isClusterImportBefore(cluster *types.Cluster) (bool, error) {
	return hasNamespace(cluster.KubeClient, ZCloudNamespace)
}

func createRole(cluster *types.Cluster, roleName string, role ClusterRole) error {
	cli := cluster.KubeClient
	if err := createServiceAccount(cli, roleName, ZCloudNamespace); err != nil {
		logger.Error("create service account %s failed: %s", roleName, err.Error())
		return err
	}

	if err := createClusterRole(cli, roleName, role); err != nil {
		logger.Error("create cluster role %s failed: %s", roleName, err.Error())
		return err
	}

	if err := createRoleBinding(cli, roleName, roleName, ZCloudNamespace); err != nil {
		logger.Error("create clusterRoleBinding %s failed: %s", ZCloudAdmin, err.Error())
		return err
	}

	return nil
}
