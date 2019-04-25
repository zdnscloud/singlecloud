package handler

import (
	"fmt"
	"time"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gok8s/exec"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/event"
	"github.com/zdnscloud/singlecloud/pkg/globaldns"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/servicecache"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	ZCloudNamespace = "zcloud"
	ZCloudAdmin     = "zcloud-cluster-admin"
	ZCloudReadonly  = "zcloud-cluster-readonly"

	MaxEventCount = 4096
)

type Cluster struct {
	*types.Cluster `json:",inline"`

	KubeClient   client.Client              `json:"-"`
	Executor     *exec.Executor             `json:"-"`
	EventWatcher *event.EventWatcher        `json:"-"`
	ServiceCache *servicecache.ServiceCache `json:"-"`
}

type ClusterManager struct {
	api.DefaultHandler
	clusters  []*Cluster
	globaldns string
}

func newClusterManager(globaldns string) *ClusterManager {
	return &ClusterManager{globaldns: globaldns}
}

func (m *ClusterManager) GetClusterForSubResource(obj resttypes.Object) *Cluster {
	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	return m.get(clusterID)
}

func (m *ClusterManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	inner := ctx.Object.(*types.Cluster)
	if c := m.get(inner.Name); c != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate cluster name")
	}

	cluster := &Cluster{
		Cluster: inner,
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

	stop := make(chan struct{})
	cache, err := cache.New(k8sconf, cache.Options{})
	if err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("create cache failed:%s", err.Error()))
	}
	go cache.Start(stop)
	cache.WaitForCacheSync(stop)

	executor, err := exec.New(k8sconf, cli, cache)
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

	eventWatcher, err := event.New(cache, MaxEventCount)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("add cluster event watcher:%s", err.Error()))
	}
	cluster.EventWatcher = eventWatcher

	serviceCache, err := servicecache.New(cache)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create service cache failed:%s", err.Error()))
	}
	cluster.ServiceCache = serviceCache

	if err = globaldns.Init(cache, m.globaldns); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("init globaldns failed:%s", err.Error()))
	}

	if err := initCluster(cluster); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("init cluster failed:%s", err.Error()))
	}

	cluster.SetCreationTimestamp(time.Now())
	m.clusters = append(m.clusters, cluster)
	return cluster, nil
}

func (m *ClusterManager) Get(ctx *resttypes.Context) interface{} {
	return m.get(ctx.Object.GetID())
}

func (m *ClusterManager) get(id string) *Cluster {
	for _, c := range m.clusters {
		if c.GetID() == id {
			return c
		}
	}
	return nil
}

func (m *ClusterManager) List(ctx *resttypes.Context) interface{} {
	return m.clusters
}

func initCluster(cluster *Cluster) error {
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

func isClusterImportBefore(cluster *Cluster) (bool, error) {
	return hasNamespace(cluster.KubeClient, ZCloudNamespace)
}

func createRole(cluster *Cluster, roleName string, role ClusterRole) error {
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
