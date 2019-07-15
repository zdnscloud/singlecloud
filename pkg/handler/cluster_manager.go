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

	lock          sync.Mutex
	clusters      []*Cluster
	eventBus      *pubsub.PubSub
	authorizer    *authorization.Authorizer
	authenticator *authentication.Authenticator
	ZKE           *zke.ZKE
}

func newClusterManager(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, eventBus *pubsub.PubSub) *ClusterManager {

	clusterMgr := &ClusterManager{
		authorizer:    authorizer,
		authenticator: authenticator,
		eventBus:      eventBus,
		ZKE:           zke.New(),
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
		fmt.Println(m.clusters)
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate cluster name")
	}

	cluster := &Cluster{
		Name:       inner.Name,
		CreateTime: time.Now(),
	}

	if err := m.ZKE.AddWithCreate(inner); err != nil {
		return inner, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("zke err %s", err))
	}

	stopCh := make(chan struct{})
	cluster.stopCh = stopCh
	m.clusters = append(m.clusters, cluster)

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

	if err := m.ZKE.AddWithOutCreate(cluster.Name, yaml); err != nil {
		return nil, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("zke err %s", err))
	}

	stopCh := make(chan struct{})
	cluster.stopCh = stopCh
	m.clusters = append(m.clusters, cluster)

	return cluster, nil
}

func getClusterInfo(c *Cluster, zc *zke.Cluster) (*types.Cluster, error) {
	cluster := &types.Cluster{}
	cluster.SetID(c.Name)
	cluster.SetType(types.ClusterType)
	cluster.Name = c.Name
	cluster.SetCreationTimestamp(c.CreateTime)

	switch zc.Status {
	case zke.ClusterCreateing:
		cluster.Status = types.CSCreateing
	case zke.ClusterCreateFailed:
		cluster.Status = types.CSCreateFailed
	case zke.ClusterUpateFailed:
		cluster.Status = types.CSUpdateFailed
	case zke.ClusterUpateing:
		cluster.Status = types.CSUpdateing
	default:
		cluster.Status = types.CSUnreachable
	}

	if cluster.Status == zke.ClusterCreateFailed || cluster.Status == zke.ClusterCreateing {
		return cluster, fmt.Errorf("cluster %s not yet created", c.Name)
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
	zkeCluster := m.ZKE.Get(target)
	m.lock.Unlock()
	if cluster == nil {
		return nil
	}
	info, _ := getClusterInfo(cluster, zkeCluster)
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
			info, _ := getClusterInfo(c, m.ZKE.Get(c.Name))
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
			zkeCluster := m.ZKE.Get(target)
			if zkeCluster.Status == zke.ClusterCreateing || zkeCluster.Status == zke.ClusterUpateing {
				zkeCluster.CancelFunc()
			}
			cluster = c
			m.clusters = append(m.clusters[:i], m.clusters[i+1:]...)
			m.ZKE.Delete(cluster.Name)
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
		msg := <-m.ZKE.MsgCh
		if msg.Error != nil {
			log.Errorf("ZKE %s", msg.Error)
		}
		m.setClusterAfterCreatedOrUpdated(msg)
	}
}

func (m *ClusterManager) setClusterAfterCreatedOrUpdated(zkeMsg zke.Msg) error {
	for _, c := range m.clusters {
		if c.Name == zkeMsg.ClusterName {
			m.lock.Lock()
			defer m.lock.Unlock()
			zc := m.ZKE.Get(c.Name)
			switch zkeMsg.Status {
			case zke.ClusterCreateComplete:
				zc.Status = zkeMsg.Status
				zc.State = zkeMsg.State
				c.KubeClient = zkeMsg.KubeClient
				c.K8sConfig = zkeMsg.KubeConfig
				cache, err := cache.New(c.K8sConfig, cache.Options{})
				if err != nil {
					return err
				}
				go cache.Start(c.stopCh)
				cache.WaitForCacheSync(c.stopCh)
				c.Cache = cache
				m.eventBus.Pub(AddCluster{Cluster: c}, eventbus.ClusterEvent)
			case zke.ClusterUpateComplete:
				zc.Status = zkeMsg.Status
				zc.State = zkeMsg.State
				c.KubeClient = zkeMsg.KubeClient
				c.K8sConfig = zkeMsg.KubeConfig
				m.eventBus.Pub(UpdateCluster{Cluster: c}, eventbus.ClusterEvent)
			default:
				zc.Status = zkeMsg.Status
			}
		}
	}
	return nil
}

func (m *ClusterManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if ctx.Action.Name == types.ClusterCancel {
		return m.cancelBuild(ctx)
	}
	return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
}

func (m *ClusterManager) cancelBuild(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	clusterName := ctx.Object.(*types.Cluster).GetID()

	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can cancel")
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	var cluster *Cluster
	for _, c := range m.clusters {
		if c.Name == clusterName {
			cluster = c
		}
		break
	}

	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s desn't exist", clusterName))
	}

	zkeCluster := m.ZKE.Get(cluster.Name)
	m.ZKE.Lock.Lock()
	defer m.ZKE.Lock.Unlock()
	switch zkeCluster.Status {
	case zke.ClusterCreateing:
		zkeCluster.CancelFunc()
	case zke.ClusterUpateing:
		zkeCluster.CancelFunc()
	default:
		return nil, resttypes.NewAPIError(resttypes.InvalidAction, "only cluster createing and updateing state can cancel")
	}

	return nil, nil
}
