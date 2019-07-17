package zke

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"k8s.io/client-go/rest"
)

type ZKEManager map[string]*ZKECluster

type Event struct {
	ID         string
	Status     types.ClusterStatus
	State      *core.FullState
	KubeClient client.Client
	K8sConfig  *rest.Config
}

func New() ZKEManager {
	return map[string]*ZKECluster{}
}

func (m ZKEManager) Create(c *types.Cluster, eventCh chan Event) error {
	if _, ok := m[c.Name]; ok {
		return fmt.Errorf("duplicate cluster name")
	}

	config, err := scClusterToZKEConfig(c)
	if err != nil {
		return err
	}
	zc := &ZKECluster{
		Config: config,
		State:  &core.FullState{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	zc.Cancel = cancel
	m[c.Name] = zc
	go zc.create(ctx, eventCh)

	return nil
}

func (m ZKEManager) Delete(id string) {
	_, ok := m[id]
	if !ok {
		log.Warnf("cluster %s not found to delete it's state", id)
	}
	delete(m, id)
}

func (m ZKEManager) Update(e Event) error {
	c, ok := m[e.ID]
	if !ok {
		log.Warnf("cluster %s not found to update it's zke state", e.ID)
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	switch e.Status {
	case types.CSCreateSuccess:
		c.State = e.State
		c.logCh = nil
	case types.CSCreateFailed:
		c.logCh = nil
	case types.CSUpdateSuccess:
		c.State = e.State
		c.logCh = nil
	case types.CSUpdateFailed:
		c.Config = c.State.CurrentState.ZKEConfig.DeepCopy()
		c.logCh = nil
	}
	return nil
}

func (m ZKEManager) Import(id string, yaml []byte, eventCh chan Event) (client.Client, *rest.Config, error) {
	if _, ok := m[id]; ok {
		return nil, nil, fmt.Errorf("duplicate cluster name")
	}

	zc := &ZKECluster{
		State: &core.FullState{},
	}

	if err := json.Unmarshal(yaml, zc.State); err != nil {
		return nil, nil, err
	}
	zc.State.DesiredState.CertificatesBundle = pki.TransformPEMToObject(zc.State.DesiredState.CertificatesBundle)
	zc.State.CurrentState.CertificatesBundle = pki.TransformPEMToObject(zc.State.CurrentState.CertificatesBundle)
	zc.Config = zc.State.CurrentState.ZKEConfig.DeepCopy()

	k8sConfYaml := zc.State.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConf, err := config.BuildConfig([]byte(k8sConfYaml))
	if err != nil {
		return nil, nil, err
	}

	kubeClient, err := client.New(k8sConf, client.Options{})
	if err != nil {
		return nil, nil, err
	}

	m[id] = zc

	return kubeClient, k8sConf, nil
}
