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
	clusterEventBufferCount = 100
)

type ZKEManager struct {
	PubEventCh      chan interface{}
	readyClusters   []*Cluster
	unreadyClusters []*Cluster
	db              storage.DB
	lock            sync.Mutex
	scVersion       string // add cluster singlecloud version for easy to confirm zcloud component version
	nodeListener    NodeListener
}

type NodeListener interface {
	IsStorageNode(cluster *Cluster, node string) (bool, error)
}

func New(db storage.DB, scVersion string, nl NodeListener) (*ZKEManager, error) {
	mgr := &ZKEManager{
		readyClusters:   make([]*Cluster, 0),
		unreadyClusters: make([]*Cluster, 0),
		PubEventCh:      make(chan interface{}, clusterEventBufferCount),
		db:              db,
		scVersion:       scVersion,
		nodeListener:    nl,
	}
	if err := mgr.loadDB(); err != nil {
		return mgr, err
	}
	return mgr, nil
}

func (m *ZKEManager) Create(ctx *restsource.Context) (restsource.Resource, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()

	typesCluster := ctx.Resource.(*types.Cluster)
	typesCluster.TrimFieldSpace()

	existCluster := m.get(typesCluster.Name)
	if existCluster != nil {
		return nil, resterr.NewAPIError(resterr.DuplicateResource, "duplicate cluster")
	}

	if err := validateConfigForCreate(typesCluster); err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidOption, fmt.Sprintf("cluster config validate failed %s", err))
	}

	config := genZKEConfig(typesCluster)
	state := clusterState{
		ZKEConfig:    config,
		CreateTime:   time.Now(),
		FullState:    &core.FullState{},
		IsUnvailable: true,
		ScVersion:    m.scVersion,
	}
	if err := createOrUpdateClusterFromDB(typesCluster.Name, state, m.db); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	cluster := newCluster(typesCluster.Name, types.CSCreateing)
	cluster.CreateTime = state.CreateTime
	cluster.config = config
	cluster.scVersion = m.scVersion
	m.addUnready(cluster)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cluster.cancel = cancel
	go cluster.Create(cancelCtx, state, m)
	typesCluster.SetID(typesCluster.Name)
	typesCluster.SetCreationTimestamp(state.CreateTime)
	return typesCluster, nil
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

	if err := createOrUpdateClusterFromDB(id, state, m.db); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	cluster := newCluster(id, types.CSConnecting)
	cluster.CreateTime = state.CreateTime
	cluster.config = state.ZKEConfig
	cluster.scVersion = types.ScVersionImported
	m.addUnready(cluster)

	kubeConfig := state.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	cancelCtx, cancel := context.WithCancel(context.Background())
	cluster.cancel = cancel
	go cluster.InitLoop(cancelCtx, kubeConfig, m, state)
	return nil, nil
}

func (m *ZKEManager) Update(ctx *restsource.Context) (restsource.Resource, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()

	typesCluster := ctx.Resource.(*types.Cluster)
	typesCluster.TrimFieldSpace()

	existCluster := m.get(typesCluster.Name)
	if existCluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", typesCluster.Name))
	}

	// doesn't support imported cluster update because no sshkey
	if existCluster.scVersion == types.ScVersionImported {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "doesn't support update imported cluster")
	}

	if !existCluster.CanUpdate() {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("cluster %s can't update on %s status", existCluster.Name, existCluster.getStatus()))
	}

	if err := validateConfigForUpdate(existCluster.ToTypesCluster(), typesCluster, m.nodeListener, existCluster); err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidOption, fmt.Sprintf("cluster config validate failed %s", err))
	}
	config := genZKEConfigForUpdateNodes(existCluster.config, typesCluster)
	existCluster.config = config

	state, err := getClusterFromDB(typesCluster.Name, m.db)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}
	state.ZKEConfig = config
	state.IsUnvailable = true
	if err := createOrUpdateClusterFromDB(typesCluster.Name, state, m.db); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	if existCluster.IsReady() {
		m.moveToUnready(existCluster)
	}

	if err := existCluster.fsm.Event(UpdateEvent); err != nil {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("cluster %s can not update", typesCluster.Name))
	}

	select {
	case _, ok := <-existCluster.stopCh:
		if !ok {
			m.PubEventCh <- DeleteCluster{Cluster: existCluster}
		}
	default:
		close(existCluster.stopCh)
		m.PubEventCh <- DeleteCluster{Cluster: existCluster}
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	existCluster.cancel = cancel
	go existCluster.Update(cancelCtx, state, m)

	return typesCluster, nil
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

	cluster := m.get(id)
	if cluster != nil {
		if cluster.IsReady() {
			return cluster
		}
	}
	return nil
}

func (m *ZKEManager) get(id string) *Cluster {
	for _, c := range m.readyClusters {
		if c.Name == id {
			return c
		}
	}

	for _, c := range m.unreadyClusters {
		if c.Name == id {
			return c
		}
	}
	return nil
}

func (m *ZKEManager) List() []*Cluster {
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

func (m *ZKEManager) Delete(id string) *resterr.APIError {
	m.lock.Lock()
	defer m.lock.Unlock()

	toDelete := m.get(id)
	if toDelete == nil {
		return resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", id))
	}

	if err := toDelete.fsm.Event(DeleteEvent); err != nil {
		return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("cluster %s can't delete %s", toDelete.Name, err.Error()))
	}

	select {
	case _, ok := <-toDelete.stopCh:
		if !ok {
			m.PubEventCh <- DeleteCluster{Cluster: toDelete}
		}
	default:
		close(toDelete.stopCh)
		m.PubEventCh <- DeleteCluster{Cluster: toDelete}
	}

	go toDelete.Destroy(context.TODO(), m)
	return nil
}

func (m *ZKEManager) CancelCluster(id string) (interface{}, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()

	c := m.get(id)
	if c == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", id))
	}
	if err := c.Cancel(); err != nil {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, err.Error())
	}
	return nil, nil
}

func (m *ZKEManager) MoveToReady(c *Cluster) {
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

func (m *ZKEManager) moveToUnready(c *Cluster) {
	m.unreadyClusters = append(m.unreadyClusters, c)
	for i, cluster := range m.readyClusters {
		if cluster.Name == c.Name {
			m.readyClusters = append(m.readyClusters[:i], m.readyClusters[i+1:]...)
			break
		}
	}
}

func (m *ZKEManager) loadDB() error {
	states, err := getClustersFromDB(m.db)
	if err != nil {
		return err
	}

	for k, v := range states {
		if v.IsUnvailable {
			cluster := newCluster(k, types.CSUnavailable)
			cluster.config = v.ZKEConfig
			cluster.stopCh = make(chan struct{})
			cluster.CreateTime = v.CreateTime
			cluster.scVersion = v.ScVersion
			m.addUnready(cluster)
		} else {
			cluster := newCluster(k, types.CSConnecting)
			cluster.config = v.ZKEConfig
			cluster.stopCh = make(chan struct{})
			cluster.CreateTime = v.CreateTime
			cluster.scVersion = v.ScVersion
			m.addUnready(cluster)
			ctx, cancel := context.WithCancel(context.Background())
			cluster.cancel = cancel
			kubeConfig := v.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
			go cluster.InitLoop(ctx, kubeConfig, m, v)
		}
	}
	return nil
}

func (m *ZKEManager) addUnready(c *Cluster) {
	m.unreadyClusters = append(m.unreadyClusters, c)
}

func (m *ZKEManager) SendEvent(e interface{}) {
	m.PubEventCh <- e
}

func (m *ZKEManager) GetDB() storage.DB {
	return m.db
}

func (m *ZKEManager) Remove(cluster *Cluster) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i, c := range m.readyClusters {
		if c.Name == cluster.Name {
			m.readyClusters = append(m.readyClusters[:i], m.readyClusters[i+1:]...)
			break
		}
	}
	for i, c := range m.unreadyClusters {
		if c.Name == cluster.Name {
			m.unreadyClusters = append(m.unreadyClusters[:i], m.unreadyClusters[i+1:]...)
			break
		}
	}
}
