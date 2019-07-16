package zke

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/rest"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
)

func (m ZKEManager) CreateCluster(c *types.Cluster, eventCh chan Event) error {
	if _, ok := m[c.Name]; ok {
		return fmt.Errorf("duplicate cluster name")
	}

	config, err := scClusterToZKEConfig(c)
	if err != nil {
		return err
	}
	zc := &ZKECluster{
		Config: config,
		logCh:  make(chan string, 5),
	}
	ctx, cancel := context.WithCancel(context.Background())
	zc.Cancel = cancel
	m[c.Name] = zc

	go createCluster(ctx, zc, eventCh)
	return nil
}

func (m ZKEManager) ImportCluster(id string, yaml []byte, eventCh chan Event) (client.Client, *rest.Config, error) {
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

func (m ZKEManager) UpdateForAddNode(cluster string, node *types.Node, eventCh chan Event) error {
	s, ok := m[cluster]
	if !ok {
		return fmt.Errorf("cluster %s not found to add node", cluster)
	}

	config, err := getNewConfigForAddNode(s.Config, node)
	if err != nil {
		return err
	}
	s.Config = config
	s.logCh = make(chan string, 5)

	ctx, cancel := context.WithCancel(context.Background())
	s.Cancel = cancel

	go updateCluster(ctx, s, eventCh)
	return nil
}

func (m ZKEManager) UpdateForDeleteNode(cluster string, node string, eventCh chan Event) error {
	s, ok := m[cluster]
	if !ok {
		return fmt.Errorf("cluster %s not found to add node", cluster)
	}

	config, err := getNewConfigForDeleteNode(s.Config, node)
	if err != nil {
		return err
	}
	s.Config = config
	s.logCh = make(chan string, 5)

	ctx, cancel := context.WithCancel(context.Background())
	s.Cancel = cancel

	go updateCluster(ctx, s, eventCh)
	return nil
}

func (m ZKEManager) Delete(id string) {
	_, ok := m[id]
	if !ok {
		log.Warnf("cluster %s not found to delete it's state", id)
	}
	delete(m, id)
}

/*
func printLog(logCh chan string) {
	for {
		log, ok := <-logCh
		if !ok {
			return
		}
		fmt.Printf(log)
	}
}
*/
