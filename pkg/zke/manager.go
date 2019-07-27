package zke

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/gok8s/client"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/zke/core/pki"
)

type ZKEManager struct {
	PubEventCh      chan interface{}
	readyClusters   []*Cluster
	unreadyClusters []*Cluster
	db              storage.DB
	lock            sync.Mutex
}

type tempCluster struct {
	Client  client.Client
	Cluster *types.Cluster
}

func New(db storage.DB) (*ZKEManager, error) {
	mgr := &ZKEManager{
		readyClusters:   make([]*Cluster, 0),
		unreadyClusters: make([]*Cluster, 0),
		PubEventCh:      make(chan interface{}),
		db:              db,
	}
	go mgr.loadDB()
	return mgr, nil
}

func (m *ZKEManager) Create(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	inner := ctx.Object.(*types.Cluster)
	c := m.get(inner.Name)

	if c != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate cluster")
	}

	config, err := scClusterToZKEConfig(inner)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("validate cluster %s config failed %s", inner.Name, err))
	}

	state := clusterState{
		ZKEConfig:    config,
		CreateTime:   time.Now(),
		IsUnvailable: true,
	}
	if err := createOrUpdateState(inner.Name, state, m.db); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("%s", err))
	}

	cluster := newInitialCluster(inner.Name)
	cluster.CreateTime = state.CreateTime
	cluster.config = config
	m.unreadyClusters = append(m.unreadyClusters, cluster)
	cluster.fsm.Event(CreateEvent)

	zkectx, cancel := context.WithCancel(context.Background())
	cluster.cancel = cancel
	go cluster.create(zkectx, state, m)
	return inner, nil
}

func (m *ZKEManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	inner := ctx.Object.(*types.Cluster)
	c := m.get(inner.Name)

	if c == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s desn't exist", inner.Name))
	}

	config, err := scClusterToZKEConfig(inner)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.InvalidOption, fmt.Sprintf("validate cluster %s config failed %s", inner.Name, err))
	}
	c.config = config

	state, err := getState(inner.Name, m.db)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("%s", err))
	}

	state.IsUnvailable = true
	if err := createOrUpdateState(inner.Name, state, m.db); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("%s", err))
	}

	m.moveTounready(c)
	c.fsm.Event(UpdateEvent)

	zkectx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.update(zkectx, state, m)

	return nil, nil
}

func (m *ZKEManager) Get(id string) *tempCluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	cluster := m.get(id)
	if cluster != nil {
		return &tempCluster{
			Client:  cluster.KubeClient,
			Cluster: cluster.getTypesCluster(),
		}
	}
	return nil
}

func (m *ZKEManager) GetReady(id string) *Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.getReady(id)
}

func (m *ZKEManager) ListAll() []*tempCluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	var clusters []*tempCluster
	for _, c := range m.readyClusters {
		clusters = append(clusters, &tempCluster{
			Client:  c.KubeClient,
			Cluster: c.getTypesCluster(),
		})
	}
	for _, c := range m.unreadyClusters {
		clusters = append(clusters, &tempCluster{
			Client:  c.KubeClient,
			Cluster: c.getTypesCluster(),
		})
	}
	return clusters
}

func (m *ZKEManager) ListReady() []*tempCluster {
	m.lock.Lock()
	defer m.lock.Unlock()
	var clusters []*tempCluster
	for _, c := range m.readyClusters {
		clusters = append(clusters, &tempCluster{
			Client:  c.KubeClient,
			Cluster: c.getTypesCluster(),
		})
	}
	return clusters
}

func (m *ZKEManager) Delete(id string) *resttypes.APIError {
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
				return resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("cluster %s in %s state desn't allow to delete", status))
			}
			m.unreadyClusters = append(m.unreadyClusters[:i], m.unreadyClusters[i+1:]...)
			break
		}
	}
	if toDelete == nil {
		return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s desn't exist", id))
	}
	if err := deleteState(id, m.db); err != nil {
		return resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("delete cluster %s from database failed", id, err))
	}
	return nil
}

func (m *ZKEManager) Cancel(id string) (interface{}, *resttypes.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()
	c := m.get(id)
	if c == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s desn't exist", id))
	}
	status := c.getStatus()
	if status == types.CSCreateing || status == types.CSUpdateing || status == types.CSConnecting {
		c.fsm.Event(CancelEvent, m)
		c.cancel()
		c.fsm.Event(CancelSuccessEvent, m)
		return nil, nil
	}
	return nil, resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("cluster %s in %s state, not allow cancel", id, status))
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
			m.addToUnreadyWithLock(cluster)
		} else {
			cluster := newInitialCluster(k)
			cluster.config = v.ZKEConfig
			cluster.stopCh = make(chan struct{})
			cluster.CreateTime = v.CreateTime
			m.addToUnreadyWithLock(cluster)
			cluster.fsm.Event(InitEvent)
			ctx, cancel := context.WithCancel(context.Background())
			cluster.cancel = cancel
			kubeConfig := v.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
			go cluster.initLoop(ctx, kubeConfig, m)
		}
	}
	return nil
}

func (m *ZKEManager) addToUnreadyWithLock(c *Cluster) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.unreadyClusters = append(m.unreadyClusters, c)
}

func (m *ZKEManager) sendPubEventWithLock(e interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()
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
	state.IsUnvailable = true
	return createOrUpdateState(c.Name, state, m.db)
}
