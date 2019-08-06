package handler

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/rest"

	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"
)

const (
	ZCloudNamespace = "zcloud"
	ZCloudAdmin     = "zcloud-cluster-admin"
	ZCloudReadonly  = "zcloud-cluster-readonly"
)

type Cluster struct {
	Name       string
	CreateTime time.Time
	KubeClient client.Client
	Cache      cache.Cache
	K8sConfig  *rest.Config
	stopCh     chan struct{}
}

type AddCluster struct {
	Cluster *Cluster
}

type DeleteCluster struct {
	Cluster *Cluster
}

type ClusterManager struct {
	api.DefaultHandler

	lock          sync.Mutex
	clusters      []*Cluster
	eventBus      *pubsub.PubSub
	authorizer    *authorization.Authorizer
	authenticator *authentication.Authenticator
	db            storage.DB
	Agent         *clusteragent.AgentManager
}

func newClusterManager(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, eventBus *pubsub.PubSub, agent *clusteragent.AgentManager, db storage.DB) *ClusterManager {
	return &ClusterManager{
		authorizer:    authorizer,
		authenticator: authenticator,
		eventBus:      eventBus,
		db:            db,
		Agent:         agent,
	}
}

func (m *ClusterManager) GetDB() storage.DB {
	return m.db
}

func (m *ClusterManager) GetAuthorizer() *authorization.Authorizer {
	return m.authorizer
}

func (m *ClusterManager) GetClusterForSubResource(obj resttypes.Object) *Cluster {
	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.get(clusterID)
}

func (m *ClusterManager) GetClusterByName(name string) *Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.get(name)
}

func (m *ClusterManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

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
	var options client.Options
	options.Scheme = client.GetDefaultScheme()
	storagev1.AddToScheme(options.Scheme)

	cli, err := client.New(k8sconf, options)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("connect to cluster failed:%s", err.Error()))
	}
	cluster.KubeClient = cli

	stopCh := make(chan struct{})
	cache, err := cache.New(k8sconf, cache.Options{})
	if err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("create cache failed:%s", err.Error()))
	}
	go cache.Start(stopCh)
	cache.WaitForCacheSync(stopCh)

	cluster.Cache = cache
	cluster.K8sConfig = k8sconf
	cluster.stopCh = stopCh
	m.clusters = append(m.clusters, cluster)

	c, err := getClusterInfo(cluster)
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get cluster info:%s", err.Error()))
	}

	m.eventBus.Pub(AddCluster{Cluster: cluster}, eventbus.ClusterEvent)
	return c, nil
}

func getClusterInfo(c *Cluster) (*types.Cluster, error) {
	cluster := &types.Cluster{}
	cluster.SetID(c.Name)
	cluster.SetType(types.ClusterType)
	cluster.Name = c.Name
	cluster.Status = types.CSUnreachable
	cluster.SetCreationTimestamp(c.CreateTime)

	version, err := c.KubeClient.ServerVersion()
	if err != nil {
		return cluster, err
	}

	cluster.Version = version.GitVersion

	nodes, err := getNodes(c.KubeClient)
	if err != nil {
		return cluster, err
	}
	cluster.NodesCount = len(nodes)
	for _, n := range nodes {
		if n.HasRole(types.RoleControlPlane) {
			continue
		}
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
	cluster.Status = types.CSRunning
	return cluster, nil
}

func (m *ClusterManager) Get(ctx *resttypes.Context) interface{} {
	target := ctx.Object.GetID()
	if m.authorizer.Authorize(getCurrentUser(ctx), target, "") == false {
		return nil
	}

	m.lock.Lock()
	cluster := m.get(target)
	m.lock.Unlock()
	if cluster == nil {
		return nil
	}
	info, _ := getClusterInfo(cluster)
	return info
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

	m.lock.Lock()
	defer m.lock.Unlock()
	for _, c := range m.clusters {
		if m.authorizer.Authorize(user, c.Name, "") {
			info, _ := getClusterInfo(c)
			clusters = append(clusters, info)
		}
	}
	return clusters
}

func (m *ClusterManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}

	target := ctx.Object.(*types.Cluster).GetID()
	m.lock.Lock()
	var cluster *Cluster
	for i, c := range m.clusters {
		if c.Name == target {
			cluster = c
			m.clusters = append(m.clusters[:i], m.clusters[i+1:]...)
			break
		}
	}
	m.lock.Unlock()

	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s desn't exist", target))
	}
	m.eventBus.Pub(DeleteCluster{Cluster: cluster}, eventbus.ClusterEvent)
	close(cluster.stopCh)
	return nil
}

func (m *ClusterManager) authorizationHandler() api.HandlerFunc {
	return func(ctx *resttypes.Context) *resttypes.APIError {
		if ctx.Object.GetType() == types.UserType {
			if ctx.Action != nil && ctx.Action.Name == types.ActionLogin {
				return nil
			}
		}

		user := getCurrentUser(ctx)
		if user == "" {
			return resttypes.NewAPIError(resttypes.Unauthorized, fmt.Sprintf("user is unknowned"))
		}

		if m.authorizer.GetUser(user) == nil {
			newUser := &types.User{Name: user}
			newUser.SetID(user)
			m.authorizer.AddUser(newUser)
		}

		ancestors := resttypes.GetAncestors(ctx.Object)
		if len(ancestors) < 2 {
			return nil
		}

		if ancestors[0].GetType() == types.ClusterType && ancestors[1].GetType() == types.NamespaceType {
			cluster := ancestors[0].GetID()
			namespace := ancestors[1].GetID()
			if m.authorizer.Authorize(user, cluster, namespace) == false {
				return resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("user %s has no sufficient permission to work on cluster %s namespace %s", user, cluster, namespace))
			}
		}
		return nil
	}
}
