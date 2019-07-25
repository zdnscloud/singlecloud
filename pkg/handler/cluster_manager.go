package handler

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/rest"

	"github.com/zdnscloud/cement/fsm"
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
	fsm        *fsm.FSM
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
	zkeManager      *zke.ZKEManager
	db              storage.DB
}

func newClusterManager(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, eventBus *pubsub.PubSub, db storage.DB) *ClusterManager {

	clusterMgr := &ClusterManager{
		authorizer:    authorizer,
		authenticator: authenticator,
		eventBus:      eventBus,
		db:            db,
	}
	zkeMgr, err := zke.New(db)
	if err != nil {
		return clusterMgr
	}
	clusterMgr.zkeManager = zkeMgr
	go clusterMgr.eventLoop()
	return clusterMgr
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
	return m.getReady(clusterID)
}

func (m *ClusterManager) GetClusterByName(name string) *Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.getReady(name)
}

func (m *ClusterManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	inner := ctx.Object.(*types.Cluster)
	if c := m.getReady(inner.Name); c != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate cluster name")
	}

	cluster := newClusterWithFsm(inner.Name, string(types.CSInit))
	cluster.CreateTime = time.Now()
	cluster.fsm.Event(zke.CreateEvent, m)

	inner.SetID(inner.Name)
	inner.SetType(types.ClusterType)
	inner.SetCreationTimestamp(cluster.CreateTime)

	if err := m.zkeManager.CreateCluster(inner); err != nil {
		return inner, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("zke err %s", err))
	}
	return inner, nil
}

func (m *ClusterManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create cluster")
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	inner := ctx.Object.(*types.Cluster)
	c := m.get(inner.Name)

	if c == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s desn't exist", inner.Name))
	}
	c.fsm.Event(zke.UpdateEvent, m)

	if err := m.zkeManager.UpdateCluster(inner); err != nil {
		return inner, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("zke err %s", err))
	}
	return inner, nil
}

func (m *ClusterManager) getReadyClusterInfo(c *Cluster) (*types.Cluster, error) {
	cluster := zke.ZKEClusterToSCCluster(m.zkeManager.GetCluster(c.Name))
	cluster.SetID(c.Name)
	cluster.SetType(types.ClusterType)
	cluster.SetCreationTimestamp(c.CreateTime)
	cluster.Status = types.CSUnreachable

	version, err := c.KubeClient.ServerVersion()
	if err != nil {
		c.fsm.Event(zke.GetInfoFailedEvent, m)
		return cluster, err
	}
	c.fsm.Event(zke.GetInfoSuccessEvent, m)
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

func (m *ClusterManager) getUnreadyClusterInfo(c *Cluster) *types.Cluster {
	cluster := zke.ZKEClusterToSCCluster(m.zkeManager.GetCluster(c.Name))
	cluster.SetID(c.Name)
	cluster.SetType(types.ClusterType)
	cluster.SetCreationTimestamp(c.CreateTime)
	cluster.Status = types.ClusterStatus(c.fsm.Current())
	return cluster
}

func (m *ClusterManager) Get(ctx *resttypes.Context) interface{} {
	target := ctx.Object.GetID()
	if m.authorizer.Authorize(getCurrentUser(ctx), target, "") == false {
		return nil
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	cluster := m.getReady(target)
	if cluster != nil {
		info, _ := m.getReadyClusterInfo(cluster)
		return info
	}

	cluster = m.getUnready(target)
	if cluster != nil {
		info := m.getUnreadyClusterInfo(cluster)
		return info
	}
	return nil
}

func (m *ClusterManager) List(ctx *resttypes.Context) interface{} {
	requestFlags := ctx.Request.URL.Query()
	user := getCurrentUser(ctx)
	var clusters []*types.Cluster

	m.lock.Lock()
	defer m.lock.Unlock()
	for _, c := range m.readyClusters {
		if m.authorizer.Authorize(user, c.Name, "") {
			info, _ := m.getReadyClusterInfo(c)
			clusters = append(clusters, info)
		}
	}

	if onlyReady := requestFlags.Get("onlyready"); onlyReady == "true" {
		return clusters
	}

	for _, c := range m.unReadyClusters {
		if m.authorizer.Authorize(user, c.Name, "") {
			info := m.getUnreadyClusterInfo(c)
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
	defer m.lock.Unlock()
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
			if cluster.fsm.Current() == string(types.CSCreateing) || cluster.fsm.Current() == string(types.CSUpdateing) || cluster.fsm.Current() == string(types.CSConnecting) {
				return resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("cluster in %s state desn't allow to delete", cluster.fsm.Current()))
			}
			m.unReadyClusters = append(m.unReadyClusters[:i], m.unReadyClusters[i+1:]...)
			break
		}
	}

	if err := m.zkeManager.DeleteCluster(target); err != nil {
		return resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}

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

func (m *ClusterManager) eventLoop() {
	for {
		e := <-m.zkeManager.EventCh
		m.lock.Lock()
		if e.Type == zke.InitEvent {
			if e.IsUnavailable {
				c := newClusterWithFsm(e.ClusterID, string(types.CSUnavailable))
				m.addToUnreadyFromEvent(c, e)
			} else {
				c := newClusterWithFsm(e.ClusterID, string(types.CSInit))
				c.fsm.Event(zke.InitEvent, m, e)
			}
		} else {
			c := m.get(e.ClusterID)
			if c == nil {
				log.Errorf("recive %s event but cluster %s desn't exist", e.Type, e.ClusterID)
				continue
			}
			c.fsm.Event(e.Type, m, e)
		}
		m.lock.Unlock()
	}
}

func (m *ClusterManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if ctx.Action.Name == types.ClusterCancel {
		target := ctx.Object.(*types.Cluster).GetID()
		cluster := m.getUnready(target)
		currentStatus := cluster.fsm.Current()
		if currentStatus == string(types.CSConnecting) || currentStatus == string(types.CSCreateing) || currentStatus == string(types.CSUpdateing) {
			cluster.fsm.Event(zke.CancelEvent, m)
			return nil, nil
		}
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("cluster %s in %s state, not allow cancel", cluster.Name, currentStatus))

	}
	return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
}

func newClusterWithFsm(name string, initStatus string) *Cluster {
	cluster := &Cluster{
		Name:   name,
		stopCh: make(chan struct{}),
	}
	fsm := fsm.NewFSM(
		initStatus,
		fsm.Events{
			{Name: zke.InitEvent, Src: []string{string(types.CSInit)}, Dst: string(types.CSConnecting)},
			{Name: zke.InitSuccessEvent, Src: []string{string(types.CSConnecting)}, Dst: string(types.CSRunning)},
			{Name: zke.CreateEvent, Src: []string{string(types.CSInit)}, Dst: string(types.CSCreateing)},
			{Name: zke.CreateSuccessEvent, Src: []string{string(types.CSCreateing)}, Dst: string(types.CSRunning)},
			{Name: zke.CreateFailedEvent, Src: []string{string(types.CSCreateing)}, Dst: string(types.CSUnavailable)},
			{Name: zke.UpdateEvent, Src: []string{string(types.CSRunning), string(types.CSUnavailable)}, Dst: string(types.CSUpdateing)},
			{Name: zke.UpdateSuccessEvent, Src: []string{string(types.CSUpdateing)}, Dst: string(types.CSRunning)},
			{Name: zke.UpdateFailedEvent, Src: []string{string(types.CSUpdateing)}, Dst: string(types.CSUnavailable)},
			{Name: zke.GetInfoFailedEvent, Src: []string{string(types.CSRunning)}, Dst: string(types.CSUnreachable)},
			{Name: zke.GetInfoSuccessEvent, Src: []string{string(types.CSUnreachable)}, Dst: string(types.CSRunning)},
			{Name: zke.CancelEvent, Src: []string{string(types.CSUpdateing), string(types.CSCreateing), string(types.CSConnecting)}, Dst: string(types.CSCanceling)},
			{Name: zke.CancelSuccessEvent, Src: []string{string(types.CSCanceling)}, Dst: string(types.CSUnavailable)},
			{Name: zke.DeleteEvent, Src: []string{string(types.CSRunning), string(types.CSUnavailable), string(types.CSUnreachable)}, Dst: string(types.CSDestroy)},
		},
		fsm.Callbacks{
			zke.InitEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ClusterManager)
				ze := e.Args[1].(zke.Event)
				mgr.addToUnreadyFromEvent(cluster, ze)
			},
			zke.InitSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ClusterManager)
				ze := e.Args[1].(zke.Event)
				mgr.pubAddFromEvent(cluster, ze)
			},
			zke.CreateEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ClusterManager)
				mgr.unReadyClusters = append(mgr.unReadyClusters, cluster)
			},
			zke.CreateSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ClusterManager)
				ze := e.Args[1].(zke.Event)
				mgr.zkeManager.UpdateClusterState(ze)
				mgr.pubAddFromEvent(cluster, ze)
			},
			zke.CreateFailedEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ClusterManager)
				ze := e.Args[1].(zke.Event)
				if err := mgr.zkeManager.UpdateClusterState(ze); err != nil {
					log.Infof("%s", err)
				}
			},
			zke.UpdateEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ClusterManager)
				mgr.moveClusterToUnready(cluster)
			},
			zke.UpdateSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ClusterManager)
				ze := e.Args[1].(zke.Event)
				mgr.zkeManager.UpdateClusterState(ze)
				mgr.pubUpdateFromEvent(cluster, ze)
			},
			zke.UpdateFailedEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ClusterManager)
				ze := e.Args[1].(zke.Event)
				mgr.zkeManager.UpdateClusterState(ze)
			},
			zke.CancelEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ClusterManager)
				mgr.zkeManager.GetCluster(cluster.Name).Cancel(mgr.zkeManager.EventCh)
			},
		},
	)
	cluster.fsm = fsm
	return cluster
}

func (m *ClusterManager) addToUnreadyFromEvent(c *Cluster, e zke.Event) {
	c.CreateTime = e.CreateTime
	m.unReadyClusters = append(m.unReadyClusters, c)
}

func (m *ClusterManager) pubAddFromEvent(c *Cluster, e zke.Event) {
	c.K8sConfig = e.K8sConfig
	c.KubeClient = e.KubeClient
	cache, err := cache.New(c.K8sConfig, cache.Options{})
	if err != nil {
		log.Errorf("build cluster %s cache err %s", c.Name, err)
	}
	go cache.Start(c.stopCh)
	cache.WaitForCacheSync(c.stopCh)
	c.Cache = cache
	m.moveClusterToready(c)
	m.eventBus.Pub(AddCluster{Cluster: c}, eventbus.ClusterEvent)
}

func (m *ClusterManager) pubUpdateFromEvent(c *Cluster, e zke.Event) {
	c.K8sConfig = e.K8sConfig
	c.KubeClient = e.KubeClient
	m.moveClusterToready(c)
	m.eventBus.Pub(UpdateCluster{Cluster: c}, eventbus.ClusterEvent)
}

func (m *ClusterManager) moveClusterToready(cluster *Cluster) {
	m.readyClusters = append(m.readyClusters, cluster)
	for i, c := range m.unReadyClusters {
		if c.Name == cluster.Name {
			m.unReadyClusters = append(m.unReadyClusters[:i], m.unReadyClusters[i+1:]...)
			break
		}
	}
}

func (m *ClusterManager) moveClusterToUnready(cluster *Cluster) {
	m.unReadyClusters = append(m.unReadyClusters, cluster)
	for i, c := range m.readyClusters {
		if c.Name == cluster.Name {
			m.readyClusters = append(m.readyClusters[:i], m.readyClusters[i+1:]...)
			break
		}
	}
}

func (m *ClusterManager) getUnready(id string) *Cluster {
	for _, c := range m.unReadyClusters {
		if c.Name == id {
			return c
		}
	}
	return nil
}

func (m *ClusterManager) getReady(id string) *Cluster {
	for _, c := range m.readyClusters {
		if c.Name == id {
			return c
		}
	}
	return nil
}

func (m *ClusterManager) get(id string) *Cluster {
	readyCluster := m.getReady(id)
	if readyCluster != nil {
		return readyCluster
	}
	return m.getUnready(id)
}
