package zke

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"k8s.io/client-go/rest"
)

const (
	ZKEManagerTable = "zke_manager"
)

type ZKEManager map[string]*ZKECluster

type Event struct {
	ID         string //cluster name
	Status     types.ClusterStatus
	State      State
	KubeClient client.Client
	K8sConfig  *rest.Config
}

func (m ZKEManager) Create(c *types.Cluster, eventCh chan Event, db storage.DB) error {
	if _, ok := m[c.Name]; ok {
		return fmt.Errorf("duplicate cluster name")
	}

	zc, err := scClusterToZKECluster(c)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	zc.cancel = cancel
	m[c.Name] = zc

	go zc.create(ctx, eventCh, db)
	return nil
}

func (m ZKEManager) Import(createTime time.Time, yaml []byte, eventCh chan Event, db storage.DB) (client.Client, *rest.Config, error) {
	zc := &ZKECluster{
		CreateTime: createTime,
	}

	s := State{
		FullState:  &core.FullState{},
		CreateTime: createTime,
	}

	if err := json.Unmarshal(yaml, s.FullState); err != nil {
		return nil, nil, err
	}
	s.DesiredState.CertificatesBundle = pki.TransformPEMToObject(s.DesiredState.CertificatesBundle)
	s.CurrentState.CertificatesBundle = pki.TransformPEMToObject(s.CurrentState.CertificatesBundle)
	zc.ZKEConfig = s.CurrentState.ZKEConfig.DeepCopy()

	k8sConfYaml := s.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConf, err := config.BuildConfig([]byte(k8sConfYaml))
	if err != nil {
		return nil, nil, err
	}

	kubeClient, err := client.New(k8sConf, client.Options{})
	if err != nil {
		return nil, nil, err
	}

	if _, ok := m[zc.ClusterName]; ok {
		return kubeClient, k8sConf, fmt.Errorf("duplicate cluster name")
	}

	m[zc.ClusterName] = zc
	if err := zc.updateOrCreateState(db, s); err != nil {
		return kubeClient, k8sConf, err
	}
	return kubeClient, k8sConf, nil
}

func (m ZKEManager) Get(id string) *ZKECluster {
	c, ok := m[id]
	if ok {
		return c
	}
	return nil
}

func (m ZKEManager) Delete(id string, db storage.DB) error {
	_, ok := m[id]
	if !ok {
		log.Warnf("cluster %s not found to delete it's state", id)
		return nil
	}
	if err := m.Get(id).deleteState(db); err != nil {
		return err
	}
	delete(m, id)
	return nil
}

func (m ZKEManager) UpdateFromEvent(e Event, db storage.DB) error {
	c, ok := m[e.ID]
	if !ok {
		log.Warnf("cluster %s not found to update it's zke state", e.ID)
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	switch e.Status {
	case types.CSCreateSuccess:
		if err := c.updateOrCreateState(db, e.State); err != nil {
			return err
		}
		c.logCh = nil
	case types.CSCreateFailed:
		c.logCh = nil
	case types.CSUpdateSuccess:
		if err := c.updateOrCreateState(db, e.State); err != nil {
			return err
		}
		c.logCh = nil
	case types.CSUpdateFailed:
		config, err := c.getConfigFromDB(db)
		if err != nil {
			return err
		}
		c.ZKEConfig = config
		c.logCh = nil
	}
	return nil
}

func New(db storage.DB) (ZKEManager, error) {
	mgr := map[string]*ZKECluster{}
	table, err := db.CreateOrGetTable(ZKEManagerTable)
	if err != nil {
		return mgr, fmt.Errorf("get table failed %s", err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return mgr, fmt.Errorf("begin transaction failed %s", err.Error())
	}

	defer tx.Commit()

	values, err := tx.List()
	if err != nil {
		return mgr, fmt.Errorf("list cluster state failed %s", err.Error())
	}

	for k, v := range values {
		s := State{}
		if err := json.Unmarshal(v, &s); err != nil {
			return mgr, fmt.Errorf("unmarshal cluster %s state failed %s", k, err.Error())
		}
		s.DesiredState.CertificatesBundle = pki.TransformPEMToObject(s.DesiredState.CertificatesBundle)
		s.CurrentState.CertificatesBundle = pki.TransformPEMToObject(s.CurrentState.CertificatesBundle)
		zc := &ZKECluster{
			ZKEConfig:  s.CurrentState.ZKEConfig.DeepCopy(),
			CreateTime: s.CreateTime,
		}
		mgr[k] = zc
	}
	return mgr, nil
}
