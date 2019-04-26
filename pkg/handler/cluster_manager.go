package handler

import (
	"fmt"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gok8s/exec"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/event"
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
	Name         string
	CreateTime   time.Time
	KubeClient   client.Client
	Executor     *exec.Executor
	EventWatcher *event.EventWatcher
	ServiceCache *servicecache.ServiceCache
	AgentManager *clusteragent.AgentManager
}

type ClusterManager struct {
	api.DefaultHandler
	clusters []*Cluster
}

func newClusterManager() *ClusterManager {
	return &ClusterManager{}
}

func (m *ClusterManager) GetClusterForSubResource(obj resttypes.Object) *Cluster {
	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	return m.get(clusterID)
}

func (m *ClusterManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}

	inner := ctx.Object.(*types.Cluster)
	if c := m.get(inner.Name); c != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate cluster name")
	}

	cluster := &Cluster{
		Name:       inner.Name,
		CreateTime: time.Now(),
	}

	k8sconf, err := config.BuildConfig(yamlConf)
	if err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("invalid cluster config:%s", err.Error()))
	}

	cli, err := client.New(k8sconf, client.Options{})
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("connect to cluster failed:%s", err.Error()))
	}
	cluster.KubeClient = cli

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

	cluster.AgentManager = clusteragent.New()
	if err := initCluster(cluster); err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("init cluster failed:%s", err.Error()))
	}

	m.clusters = append(m.clusters, cluster)

	c, err := getClusterInfo(cluster)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get cluster info:%s", err.Error()))
	}
	return c, nil
}

func getClusterInfo(c *Cluster) (*types.Cluster, error) {
	cluster := &types.Cluster{}
	cluster.SetID(c.Name)
	cluster.Name = c.Name

	version, err := c.KubeClient.ServerVersion()
	if err != nil {
		return nil, err
	}
	cluster.Version = version.GitVersion

	nodes, err := getNodes(c.KubeClient)
	if err != nil {
		return nil, err
	}

	cluster.NodesCount = len(nodes)
	for _, n := range nodes {
		cluster.Cpu += n.Cpu
		cluster.CpuUsed += n.CpuUsed
		cluster.Memory += n.Memory
		cluster.MemoryUsed += n.MemoryUsed
		cluster.Pod += n.Pod
		cluster.PodUsed += n.PodUsed
	}
	cluster.CpuUsedRatio = fmt.Sprintf("%.2f", float64(cluster.CpuUsed)/float64(cluster.Cpu))
	cluster.MemoryUsedRatio = fmt.Sprintf("%.2f", float64(cluster.MemoryUsed)/float64(cluster.Memory))
	cluster.PodUsedRatio = fmt.Sprintf("%.2f", float64(cluster.PodUsed)/float64(cluster.Pod))
	cluster.SetCreationTimestamp(c.CreateTime)
	return cluster, nil
}

func (m *ClusterManager) Get(ctx *resttypes.Context) interface{} {
	target := ctx.Object.GetID()
	if hasClusterPermission(getCurrentUser(ctx), target) == false {
		return nil
	} else {
		cluster := m.get(target)
		if cluster == nil {
			return nil
		}
		info, err := getClusterInfo(cluster)
		if err == nil {
			return info
		} else {
			return nil
		}
	}
}

func (m *ClusterManager) get(id string) *Cluster {
	for _, c := range m.clusters {
		if c.Name == id {
			return c
		}
	}
	return nil
}

func (m *ClusterManager) List(ctx *resttypes.Context) interface{} {
	user := getCurrentUser(ctx)
	var clusters []*types.Cluster
	for _, c := range m.clusters {
		if hasClusterPermission(user, c.Name) {
			info, err := getClusterInfo(c)
			if err == nil {
				clusters = append(clusters, info)
			}
		}
	}
	return clusters
}

func initCluster(cluster *Cluster) error {
	imported, err := isClusterImportBefore(cluster)
	if err != nil {
		return err
	}
	if imported {
		log.Infof("cluster %s has been imported before", cluster.Name)
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
		log.Errorf("create service account %s failed: %s", roleName, err.Error())
		return err
	}

	if err := createClusterRole(cli, roleName, role); err != nil {
		log.Errorf("create cluster role %s failed: %s", roleName, err.Error())
		return err
	}

	if err := createRoleBinding(cli, roleName, roleName, ZCloudNamespace); err != nil {
		log.Errorf("create clusterRoleBinding %s failed: %s", ZCloudAdmin, err.Error())
		return err
	}

	return nil
}
