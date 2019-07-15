package zke

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
)

func (z *ZKE) AddWithCreate(c *types.Cluster) error {
	z.Lock.Lock()
	defer z.Lock.Unlock()
	if _, ok := z.Clusters[c.Name]; ok {
		return fmt.Errorf("duplicate cluster name")
	}

	config, err := scClusterToZKEConfig(c)
	if err != nil {
		return err
	}
	cluster := &Cluster{
		Config: config,
		Status: ClusterCreateing,
		logCh:  make(chan string, 5),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cluster.CancelFunc = cancel
	z.Clusters[c.Name] = cluster
	go createCluster(ctx, cluster, z.MsgCh)
	return nil
}

func (z *ZKE) AddWithOutCreate(c string, yaml []byte) error {
	z.Lock.Lock()
	defer z.Lock.Unlock()
	if _, ok := z.Clusters[c]; ok {
		return fmt.Errorf("duplicate cluster name")
	}

	cluster := &Cluster{
		State:  &core.FullState{},
		Status: ClusterCreateComplete,
	}

	if err := json.Unmarshal(yaml, cluster.State); err != nil {
		return err
	}
	cluster.State.DesiredState.CertificatesBundle = pki.TransformPEMToObject(cluster.State.DesiredState.CertificatesBundle)
	cluster.State.CurrentState.CertificatesBundle = pki.TransformPEMToObject(cluster.State.CurrentState.CertificatesBundle)
	cluster.Config = cluster.State.CurrentState.ZKEConfig.DeepCopy()

	k8sConfYaml := cluster.State.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConf, err := config.BuildConfig([]byte(k8sConfYaml))
	if err != nil {
		return err
	}

	k8sClient, err := client.New(k8sConf, client.Options{})
	if err != nil {
		return err
	}

	z.Clusters[c] = cluster

	msg := Msg{
		ClusterName: c,
		KubeConfig:  k8sConf,
		KubeClient:  k8sClient,
		Status:      cluster.Status,
		State:       cluster.State,
	}
	z.MsgCh <- msg

	return nil
}

func (z *ZKE) UpdateForAddNode(cluster string, node *types.Node) error {
	z.Lock.Lock()
	defer z.Lock.Unlock()
	c, ok := z.Clusters[cluster]
	if !ok {
		return fmt.Errorf("cluster %s not found to add node", cluster)
	}

	config, err := getNewConfigForAddNode(c.Config, node)
	if err != nil {
		return err
	}
	c.Config = config
	c.Status = ClusterUpateing
	c.logCh = make(chan string, 5)

	ctx, cancel := context.WithCancel(context.Background())
	c.CancelFunc = cancel

	go updateCluster(ctx, c, z.MsgCh)
	return nil
}

func (z *ZKE) UpdateForDeleteNode(cluster string, node string) error {
	z.Lock.Lock()
	defer z.Lock.Unlock()
	c, ok := z.Clusters[cluster]
	if !ok {
		return fmt.Errorf("cluster %s not found to add node", cluster)
	}

	config, err := getNewConfigForDeleteNode(c.Config, node)
	if err != nil {
		return err
	}
	c.Config = config
	c.Status = ClusterUpateing
	c.logCh = make(chan string, 5)

	ctx, cancel := context.WithCancel(context.Background())
	c.CancelFunc = cancel

	go updateCluster(ctx, c, z.MsgCh)
	return nil
}

func (z *ZKE) Delete(clusterID string) error {
	z.Lock.Lock()
	defer z.Lock.Unlock()
	_, ok := z.Clusters[clusterID]
	if !ok {
		return fmt.Errorf("zke cluster %s not exist", clusterID)
	}
	delete(z.Clusters, clusterID)
	return nil
}

func (z *ZKE) Get(clusterID string) *Cluster {
	c, ok := z.Clusters[clusterID]
	if !ok {
		return nil
	}
	return c
}
