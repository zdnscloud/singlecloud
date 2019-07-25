package zke

import (
	"context"
	"fmt"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/zke/core/pki"
	"k8s.io/client-go/rest"
)

const (
	ZKEManagerTable = "zke_manager"

	InitEvent           = "init"
	InitSuccessEvent    = "initSuccess"
	CreateEvent         = "create"
	CreateSuccessEvent  = "createSuccess"
	CreateFailedEvent   = "createFailed"
	UpdateEvent         = "update"
	UpdateSuccessEvent  = "updateSuccess"
	UpdateFailedEvent   = "updateFailed"
	GetInfoFailedEvent  = "getInfoFailed"
	GetInfoSuccessEvent = "getInfoSuccess"
	CancelEvent         = "cancel"
	CancelSuccessEvent  = "cancelSuccess"
	DeleteEvent         = "delete"
)

type EventType string

type ZKEManager struct {
	clusters map[string]*ZKECluster
	EventCh  chan Event
	db       storage.DB
}

type Event struct {
	Type          string
	ClusterID     string
	IsUnavailable bool
	State         State
	KubeClient    client.Client
	K8sConfig     *rest.Config
	CreateTime    time.Time
}

func (m *ZKEManager) CreateCluster(c *types.Cluster) error {
	if c := m.GetCluster(c.Name); c != nil {
		return fmt.Errorf("duplicate cluster name")
	}
	zc, err := scClusterToZKECluster(c)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	zc.cancel = cancel
	m.clusters[c.Name] = zc

	state := State{
		ZKEConfig:    zc.ZKEConfig,
		CreateTime:   zc.CreateTime,
		IsUnvailable: true,
	}
	if err := m.createOrUpdateState(c.Name, state); err != nil {
		return err
	}

	go zc.create(ctx, m.EventCh)
	return nil
}

func (m *ZKEManager) UpdateCluster(c *types.Cluster) error {
	oldZC := m.GetCluster(c.Name)
	if oldZC == nil {
		return fmt.Errorf("cluster %s desn't exist in zke-manager", c.Name)
	}
	newZC, err := scClusterToZKECluster(c)
	if err != nil {
		return err
	}
	oldZC = newZC

	ctx, cancel := context.WithCancel(context.Background())
	newZC.cancel = cancel
	state, err := m.getState(c.Name)
	state.IsUnvailable = true
	if err := m.createOrUpdateState(c.Name, state); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	go newZC.update(ctx, state, m.EventCh)
	return nil
}

func (m *ZKEManager) GetCluster(id string) *ZKECluster {
	c, ok := m.clusters[id]
	if ok {
		return c
	}
	return nil
}

func (m *ZKEManager) DeleteCluster(id string) error {
	c := m.GetCluster(id)
	if c == nil {
		log.Warnf("cluster %s desn't exist to delete it's state", id)
		return nil
	}
	if err := m.deleteState(id); err != nil {
		return err
	}
	delete(m.clusters, id)
	return nil
}

func (m *ZKEManager) UpdateClusterState(e Event) error {
	c := m.GetCluster(e.ClusterID)
	if c == nil {
		log.Warnf("cluster %s desn't exist to update it's state", e.ClusterID)
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logCh = nil
	return m.createOrUpdateState(e.ClusterID, e.State)
}

func New(db storage.DB) (*ZKEManager, error) {
	mgr := &ZKEManager{
		clusters: make(map[string]*ZKECluster),
		EventCh:  make(chan Event),
		db:       db,
	}
	go mgr.initFromDB()
	return mgr, nil
}

func (m *ZKEManager) initFromDB() {
	stateMap, err := m.listState()
	if err != nil {
		log.Fatalf("%s", err)
	}

	for k, v := range stateMap {
		cluster := &ZKECluster{
			CreateTime: v.CreateTime,
			ZKEConfig:  v.ZKEConfig,
		}

		event := Event{
			ClusterID:  k,
			Type:       InitEvent,
			CreateTime: v.CreateTime,
		}
		m.clusters[k] = cluster

		if v.IsUnvailable {
			event.IsUnavailable = true
			m.EventCh <- event
			continue
		}
		m.EventCh <- event
		ctx, cancel := context.WithCancel(context.Background())
		cluster.cancel = cancel
		kubeConfig := v.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
		go cluster.initProcess(ctx, kubeConfig, m.EventCh)
	}
}
