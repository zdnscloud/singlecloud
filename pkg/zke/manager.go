package zke

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	resterr "github.com/zdnscloud/gorest/error"
	restsource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
)

const (
	PubEventBufferCount = 500
)

type ZKEManager struct {
	PubEventCh      chan interface{}
	readyClusters   []*Cluster
	unreadyClusters []*Cluster
	db              storage.DB
	lock            sync.Mutex
	scVersion       string // add cluster singlecloud version for easy to confirm zcloud component version
}

func New(db storage.DB, scVersion string) (*ZKEManager, error) {
	mgr := &ZKEManager{
		readyClusters:   make([]*Cluster, 0),
		unreadyClusters: make([]*Cluster, 0),
		PubEventCh:      make(chan interface{}, PubEventBufferCount),
		db:              db,
		scVersion:       scVersion,
	}
	if err := mgr.loadDB(); err != nil {
		return mgr, err
	}
	return mgr, nil
}

func (m *ZKEManager) Create(ctx *restsource.Context) (restsource.Resource, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	inner := ctx.Resource.(*types.Cluster)
	c := m.get(inner.Name)

	if c != nil {
		return nil, resterr.NewAPIError(resterr.DuplicateResource, "duplicate cluster")
	}

	if err := validateConfigForCreate(inner); err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidOption, fmt.Sprintf("cluster config validate failed %s", err))
	}

	config := scClusterToZKEConfig(inner)

	state := clusterState{
		ZKEConfig:    config,
		CreateTime:   time.Now(),
		FullState:    &core.FullState{},
		IsUnvailable: true,
		ScVersion:    m.scVersion,
	}

	if err := createOrUpdateState(inner.Name, state, m.db); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	cluster := newInitialCluster(inner.Name)
	cluster.CreateTime = state.CreateTime
	cluster.config = config
	cluster.scVersion = m.scVersion
	m.unreadyClusters = append(m.unreadyClusters, cluster)
	cluster.fsm.Event(CreateEvent)

	zkectx, cancel := context.WithCancel(context.Background())
	cluster.cancel = cancel
	go cluster.create(zkectx, state, m)
	inner.SetID(inner.Name)
	inner.SetCreationTimestamp(state.CreateTime)
	inner.SSHKey = ""
	return inner, nil
}

func (m *ZKEManager) Import(ctx *restsource.Context) (interface{}, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	id := ctx.Resource.(*types.Cluster).GetID()
	action := ctx.Resource.GetAction()

	c := m.get(id)
	if c != nil {
		return nil, resterr.NewAPIError(resterr.DuplicateResource, "duplicate cluster")
	}

	zkeFullState := action.Input.(*core.FullState)
	if zkeFullState != nil && zkeFullState.DesiredState.CertificatesBundle != nil {
		zkeFullState.DesiredState.CertificatesBundle = pki.TransformPEMToObject(zkeFullState.DesiredState.CertificatesBundle)
		zkeFullState.CurrentState.CertificatesBundle = pki.TransformPEMToObject(zkeFullState.CurrentState.CertificatesBundle)
	}

	state := clusterState{
		FullState:  zkeFullState,
		ZKEConfig:  zkeFullState.CurrentState.ZKEConfig.DeepCopy(),
		CreateTime: time.Now(),
		ScVersion:  types.ScVersionImported,
	}

	if err := createOrUpdateState(id, state, m.db); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	cluster := newClusterWithStatus(id, types.CSConnecting)
	cluster.CreateTime = state.CreateTime
	cluster.config = state.ZKEConfig
	cluster.scVersion = types.ScVersionImported
	m.unreadyClusters = append(m.unreadyClusters, cluster)

	kubeConfig := state.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	zkectx, cancel := context.WithCancel(context.Background())
	cluster.cancel = cancel
	go cluster.initLoop(zkectx, kubeConfig, state, m)
	return nil, nil
}

func (m *ZKEManager) Update(ctx *restsource.Context) (restsource.Resource, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	inner := ctx.Resource.(*types.Cluster)
	c := m.get(inner.Name)

	if c == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", inner.Name))
	}

	if !c.CanUpdate() {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("cluster can't update when it's %s status now", c.getStatus()))
	}

	if err := validateConfigForUpdate(c.ToTypesCluster(), inner); err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidOption, fmt.Sprintf("cluster config validate failed %s", err))
	}

	config := updateConfigNodesFromScCluster(c.config, inner)
	c.config = config

	state, err := getState(inner.Name, m.db)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	state.ZKEConfig = config
	state.IsUnvailable = true
	if err := createOrUpdateState(inner.Name, state, m.db); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	if cluster := m.getReady(c.Name); cluster != nil {
		m.moveTounready(c)
	}
	c.fsm.Event(UpdateEvent)

	select {
	case _, ok := <-c.stopCh:
		if !ok {
			m.sendPubEvent(DeleteCluster{Cluster: c})
		}
	default:
		close(c.stopCh)
		m.sendPubEvent(DeleteCluster{Cluster: c})
	}

	zkectx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.update(zkectx, state, m)

	return inner, nil
}

func (m *ZKEManager) Get(id string) *Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	cluster := m.get(id)
	if cluster != nil {
		return cluster
	}
	return nil
}

func (m *ZKEManager) GetReady(id string) *Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.getReady(id)
}

func (m *ZKEManager) ListAll() []*Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	var clusters []*Cluster
	for _, c := range m.readyClusters {
		clusters = append(clusters, c)
	}
	for _, c := range m.unreadyClusters {
		clusters = append(clusters, c)
	}
	return clusters
}

func (m *ZKEManager) ListReady() []*Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	var clusters []*Cluster
	for _, c := range m.readyClusters {
		clusters = append(clusters, c)
	}
	return clusters
}

func (m *ZKEManager) Delete(id string) *resterr.APIError {
	m.lock.Lock()
	defer m.lock.Unlock()
	var toDelete *Cluster
	for i, c := range m.readyClusters {
		if c.Name == id {
			toDelete = c
			m.readyClusters = append(m.readyClusters[:i], m.readyClusters[i+1:]...)
			break
		}
	}

	for i, c := range m.unreadyClusters {
		if c.Name == id {
			toDelete = c
			status := c.getStatus()
			if status == types.CSCreateing || status == types.CSUpdateing || status == types.CSConnecting || status == types.CSCanceling {
				return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("cluster %s in %s state desn't allow to delete", id, status))
			}
			m.unreadyClusters = append(m.unreadyClusters[:i], m.unreadyClusters[i+1:]...)
			break
		}
	}
	if toDelete == nil {
		return resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", id))
	}
	if err := deleteState(id, m.db); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("delete cluster %s from database failed %s", id, err))
	}

	select {
	case _, ok := <-toDelete.stopCh:
		if !ok {
			m.sendPubEvent(DeleteCluster{Cluster: toDelete})
			return nil
		}
	default:
		close(toDelete.stopCh)
	}
	m.sendPubEvent(DeleteCluster{Cluster: toDelete})
	return nil
}

func (m *ZKEManager) Cancel(id string) (interface{}, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	c := m.get(id)
	if c == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", id))
	}
	status := c.getStatus()
	if status == types.CSCreateing || status == types.CSUpdateing || status == types.CSConnecting {
		c.fsm.Event(CancelEvent, m)
		c.cancel()
		c.isCanceled = true
		return nil, nil
	}
	return nil, resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("cluster %s in %s state, not allow cancel", id, status))
}

func (m *ZKEManager) GetKubeConfig(id string) (interface{}, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	c := m.get(id)
	if c == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", id))
	}
	state, err := getState(id, m.db)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get cluster %s state from database failed %s", id, err))
	}
	if state.FullState != nil && state.FullState.DesiredState.CertificatesBundle != nil {
		kubeConfig := state.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
		return map[string]string{
			"name":   id,
			"config": kubeConfig,
		}, nil
	}
	return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("cluster %s not yet created", id))
}

func (m *ZKEManager) getReady(id string) *Cluster {
	for _, c := range m.readyClusters {
		if c.Name == id {
			return c
		}
	}
	return nil
}

func (m *ZKEManager) getUnready(id string) *Cluster {
	for _, c := range m.unreadyClusters {
		if c.Name == id {
			return c
		}
	}
	return nil
}

func (m *ZKEManager) get(id string) *Cluster {
	c := m.getReady(id)
	if c != nil {
		return c
	}
	return m.getUnready(id)
}

func (m *ZKEManager) moveToreadyWithLock(c *Cluster) {
	m.lock.Lock()
	defer m.lock.Unlock()
	c.logCh = nil
	m.readyClusters = append(m.readyClusters, c)
	for i, cluster := range m.unreadyClusters {
		if cluster.Name == c.Name {
			m.unreadyClusters = append(m.unreadyClusters[:i], m.unreadyClusters[i+1:]...)
			break
		}
	}
}

func (m *ZKEManager) moveTounready(c *Cluster) {
	m.unreadyClusters = append(m.unreadyClusters, c)
	for i, cluster := range m.readyClusters {
		if cluster.Name == c.Name {
			m.readyClusters = append(m.readyClusters[:i], m.readyClusters[i+1:]...)
			break
		}
	}
}

func (m *ZKEManager) loadDB() error {
	stateMap, err := listState(m.db)
	if err != nil {
		return err
	}

	for k, v := range stateMap {
		if v.IsUnvailable {
			cluster := newClusterWithStatus(k, types.CSUnavailable)
			cluster.config = v.ZKEConfig
			cluster.stopCh = make(chan struct{})
			cluster.CreateTime = v.CreateTime
			cluster.scVersion = v.ScVersion
			m.addToUnreadyWithLock(cluster)
		} else {
			cluster := newInitialCluster(k)
			cluster.config = v.ZKEConfig
			cluster.stopCh = make(chan struct{})
			cluster.CreateTime = v.CreateTime
			cluster.scVersion = v.ScVersion
			m.addToUnreadyWithLock(cluster)
			cluster.fsm.Event(InitEvent)
			ctx, cancel := context.WithCancel(context.Background())
			cluster.cancel = cancel
			kubeConfig := v.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
			go cluster.initLoop(ctx, kubeConfig, v, m)
		}
	}
	return nil
}

func (m *ZKEManager) addToUnreadyWithLock(c *Cluster) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.unreadyClusters = append(m.unreadyClusters, c)
}

func (m *ZKEManager) sendPubEvent(e interface{}) {
	m.PubEventCh <- e
}

func (m *ZKEManager) updateClusterStateWithLock(c *Cluster, s clusterState) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return createOrUpdateState(c.Name, s, m.db)
}

func (m *ZKEManager) setClusterUnavailable(c *Cluster) error {
	state, err := getState(c.Name, m.db)
	if err != nil {
		return err
	}
	// only connecting status cancel action need update and write db
	if state.IsUnvailable {
		return nil
	}
	state.IsUnvailable = true
	return createOrUpdateState(c.Name, state, m.db)
}
