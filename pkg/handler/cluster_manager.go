package handler

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/rest"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
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
	status     types.ClusterStatus
	stopCh     chan struct{}
}

type AddCluster struct {
	Cluster *Cluster
}

type DeleteCluster struct {
	Cluster *Cluster
}

type UpdateCluster struct {
	Cluster *Cluster
}
type ClusterManager struct {
	api.DefaultHandler

	lock            sync.Mutex
	readyClusters   []*Cluster
	unReadyClusters []*Cluster
	eventBus        *pubsub.PubSub
	authorizer      *authorization.Authorizer
	authenticator   *authentication.Authenticator
	zkeEventCh      chan zke.Event
	zkeManager      zke.ZKEManager
}

func newClusterManager(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, eventBus *pubsub.PubSub) *ClusterManager {

	clusterMgr := &ClusterManager{
		authorizer:    authorizer,
		authenticator: authenticator,
		eventBus:      eventBus,
		zkeManager:    zke.New(),
		zkeEventCh:    make(chan zke.Event),
	}
	go clusterMgr.zkeEventLoop()
	return clusterMgr
}

func (m *ClusterManager) GetClusterForSubResource(obj resttypes.Object) *Cluster {
	ancestors := resttypes.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.get(clusterID)
}

func (m *ClusterManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}

	if len(yamlConf) > 0 {
		return m.importExternalCluster(ctx, yamlConf)
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
		status:     types.CSCreateing,
	}

	stopCh := make(chan struct{})
	cluster.stopCh = stopCh
	m.unReadyClusters = append(m.unReadyClusters, cluster)

	if err := m.zkeManager.CreateCluster(inner, m.zkeEventCh); err != nil {
		return inner, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("zke err %s", err))
	}

	inner.SetID(inner.Name)
	inner.SetType(types.ClusterType)
	inner.Status = types.CSCreateing
	inner.SetCreationTimestamp(cluster.CreateTime)
	return inner, nil
}

func (m *ClusterManager) importExternalCluster(ctx *resttypes.Context, yaml []byte) (interface{}, *resttypes.APIError) {
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

	kubeClient, k8sConfig, err := m.zkeManager.ImportCluster(cluster.Name, yaml, m.zkeEventCh)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("zke err %s", err))
	}
	cluster.KubeClient = kubeClient
	cluster.K8sConfig = k8sConfig

	stopCh := make(chan struct{})
	cluster.stopCh = stopCh
	m.readyClusters = append(m.readyClusters, cluster)

	cache, err := cache.New(cluster.K8sConfig, cache.Options{})
	if err != nil {
		return cluster, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("create cache for cluster %s failed:%s", cluster.Name, err))
	}
	go cache.Start(cluster.stopCh)
	cache.WaitForCacheSync(cluster.stopCh)
	cluster.Cache = cache

	m.eventBus.Pub(AddCluster{Cluster: cluster}, eventbus.ClusterEvent)

	return cluster, nil
}

func getClusterInfo(c *Cluster) (*types.Cluster, error) {
	cluster := &types.Cluster{}
	cluster.SetID(c.Name)
	cluster.SetType(types.ClusterType)
	cluster.Name = c.Name
	cluster.SetCreationTimestamp(c.CreateTime)
	cluster.Status = types.CSUnreachable

	if len(c.status) > 0 {
		cluster.Status = c.status
	}

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
	for _, c := range m.readyClusters {
		if c.Name == id {
			return c
		}
	}
	return nil
}

func getUnreadyClusterInfo(c *Cluster) *types.Cluster {
	cluster := &types.Cluster{}
	cluster.SetID(c.Name)
	cluster.SetType(types.ClusterType)
	cluster.Name = c.Name
	cluster.SetCreationTimestamp(c.CreateTime)
	cluster.Status = c.status
	return cluster
}

func (m *ClusterManager) List(ctx *resttypes.Context) interface{} {
	requestFlags := ctx.Request.URL.Query()
	user := getCurrentUser(ctx)
	var clusters []*types.Cluster

	m.lock.Lock()
	defer m.lock.Unlock()
	for _, c := range m.readyClusters {
		if m.authorizer.Authorize(user, c.Name, "") {
			info, _ := getClusterInfo(c)
			clusters = append(clusters, info)
		}
	}

	if includeUnready := requestFlags.Get("includeUnready"); includeUnready == "true" {
		for _, c := range m.unReadyClusters {
			if m.authorizer.Authorize(user, c.Name, "") {
				info := getUnreadyClusterInfo(c)
				clusters = append(clusters, info)
			}
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
	for i, c := range m.readyClusters {
		if c.Name == target {
			cluster = c
			m.readyClusters = append(m.readyClusters[:i], m.readyClusters[i+1:]...)
			break
		}
	}

	for i, c := range m.unReadyClusters {
		if c.Name == target {
			cluster = c
			m.unReadyClusters = append(m.unReadyClusters[:i], m.unReadyClusters[i+1:]...)
			break
		}
	}

	m.zkeManager.Delete(cluster.Name)
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

func (m *ClusterManager) zkeEventLoop() {
	for {
		event := <-m.zkeEventCh
		if err := m.setClusterAfterCreatedOrUpdated(event); err != nil {
			log.Errorf("set cluster err info: %s", err)
		}
	}
}

func (m *ClusterManager) setClusterAfterCreatedOrUpdated(event zke.Event) error {
	for _, c := range m.unReadyClusters {
		if c.Name == event.ID {
			m.lock.Lock()
			defer m.lock.Unlock()
			switch event.Status {
			case types.CSCreateSuccess:
				c.KubeClient = event.KubeClient
				c.K8sConfig = event.K8sConfig
				m.zkeManager[c.Name].State = event.State
				cache, err := cache.New(c.K8sConfig, cache.Options{})
				if err != nil {
					return err
				}
				go cache.Start(c.stopCh)
				cache.WaitForCacheSync(c.stopCh)
				c.Cache = cache
				m.moveToready(c)
				m.eventBus.Pub(AddCluster{Cluster: c}, eventbus.ClusterEvent)
			case types.CSUpdateSuccess:
				c.KubeClient = event.KubeClient
				c.K8sConfig = event.K8sConfig
				m.zkeManager[c.Name].State = event.State
				m.moveToready(c)
				m.eventBus.Pub(UpdateCluster{Cluster: c}, eventbus.ClusterEvent)
			case types.CSUpdateFailed:
				c.status = event.Status
				m.moveToready(c)
			default:
				c.status = event.Status
			}
		}
	}
	return nil
}

func (m *ClusterManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if ctx.Action.Name == types.ClusterCancel {
		return m.cancel(ctx)
	}
	return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
}

func (m *ClusterManager) cancel(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	target := ctx.Object.(*types.Cluster).GetID()

	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can cancel")
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	var cluster *Cluster
	for _, c := range m.unReadyClusters {
		if c.Name == target {
			cluster = c
		}
		break
	}

	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s desn't exist", target))
	}

	zkeCluster, ok := m.zkeManager[target]
	if !ok {
		return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("zke state not found for %s to cancel", target))
	}
	if cluster.status == types.CSCreateing || cluster.status == types.CSUpdateing {
		zkeCluster.Cancel()
	}

	return nil, nil
}

func (m *ClusterManager) moveToready(cluster *Cluster) {
	m.readyClusters = append(m.readyClusters, cluster)

	for i, c := range m.unReadyClusters {
		if c.Name == cluster.Name {
			m.unReadyClusters = append(m.unReadyClusters[:i], m.unReadyClusters[i+1:]...)
			break
		}
	}
}

func (m *ClusterManager) moveToUnready(cluster *Cluster) {
	m.unReadyClusters = append(m.unReadyClusters, cluster)

	for i, c := range m.readyClusters {
		if c.Name == cluster.Name {
			m.readyClusters = append(m.readyClusters[:i], m.readyClusters[i+1:]...)
			break
		}
	}
}
