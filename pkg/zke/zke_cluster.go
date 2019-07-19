package zke

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/hack/sockjs"
	"github.com/zdnscloud/singlecloud/pkg/types"
	sctypes "github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	gok8sconfig "github.com/zdnscloud/gok8s/client/config"
	zkecmd "github.com/zdnscloud/zke/cmd"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	zketypes "github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

type ZKECluster struct {
	*zketypes.ZKEConfig
	logCh      chan string
	logSession sockjs.Session
	cancel     context.CancelFunc
	lock       sync.Mutex
	CreateTime time.Time
}

type State struct {
	*core.FullState `json:",inline"`
	CreateTime      time.Time `json:"createTime"`
}

func (zc *ZKECluster) AddNode(node *sctypes.Node, eventCh chan Event, db storage.DB) error {

	newConfig := zc.ZKEConfig.DeepCopy()

	newNode := zketypes.ZKEConfigNode{
		NodeName: node.Name,
		Address:  node.Address,
		Role:     node.Roles,
	}

	newConfig.Nodes = append(newConfig.Nodes, newNode)

	if err := validateConfig(newConfig); err != nil {
		return err
	}
	zc.ZKEConfig = newConfig

	ctx, cancel := context.WithCancel(context.Background())
	zc.cancel = cancel

	go zc.update(ctx, eventCh, db)
	return nil
}

func (zc *ZKECluster) DeleteNode(nodeID string, eventCh chan Event, db storage.DB) error {
	newConfig := zc.ZKEConfig.DeepCopy()

	for i, n := range newConfig.Nodes {
		if n.NodeName == nodeID {
			newConfig.Nodes = append(newConfig.Nodes[:i], newConfig.Nodes[i+1:]...)
		}
	}

	if err := validateConfig(newConfig); err != nil {
		return err
	}
	zc.ZKEConfig = newConfig

	ctx, cancel := context.WithCancel(context.Background())
	zc.cancel = cancel

	go zc.update(ctx, eventCh, db)
	return nil
}

func (c *ZKECluster) create(ctx context.Context, eventCh chan Event, db storage.DB) {
	var event = Event{
		ID:     c.ZKEConfig.ClusterName,
		Status: types.CSCreateFailed,
	}

	defer func(eventCh chan Event) {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			eventCh <- event
		}
	}(eventCh)

	logger, logCh := log.NewLog4jBufLogger(MaxZKELogLines, log.Info)
	defer logger.Close()
	c.logCh = logCh

	event, err := c.up(ctx, event, db)
	if err != nil {
		log.Errorf("zke err info %s", err)
		eventCh <- event
		return
	}
	event.Status = types.CSCreateSuccess
	eventCh <- event
}

func (c *ZKECluster) update(ctx context.Context, eventCh chan Event, db storage.DB) {
	var event = Event{
		ID:     c.ZKEConfig.ClusterName,
		Status: types.CSUpdateFailed,
	}

	defer func(eventCh chan Event) {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			eventCh <- event
		}
	}(eventCh)

	event, err := c.up(ctx, event, db)
	if err != nil {
		log.Errorf("zke err info %s", err)
		eventCh <- event
	}
	event.Status = types.CSUpdateSuccess
	eventCh <- event
}

func (c *ZKECluster) Cancel() {
	c.cancel()
}

func (c *ZKECluster) up(ctx context.Context, event Event, db storage.DB) (Event, error) {
	logger, logCh := log.NewLog4jBufLogger(MaxZKELogLines, log.Info)
	defer logger.Close()
	c.logCh = logCh

	state, err := c.getState(db)
	if err != nil {
		return event, err
	}

	newState, err := zkecmd.ClusterUpFromRest(ctx, c.ZKEConfig, state.FullState, logger)
	if err != nil {
		return event, err
	}

	kubeConfigYaml := newState.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConfig, err := gok8sconfig.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		return event, err
	}

	kubeClient, err := client.New(k8sConfig, client.Options{})
	if err != nil {
		return event, err
	}

	if err := deployZcloudProxy(kubeClient, c.ClusterName, c.SingleCloudAddress); err != nil {
		return event, err
	}

	event.KubeClient = kubeClient
	event.K8sConfig = k8sConfig
	event.State = State{
		FullState:  newState,
		CreateTime: c.CreateTime,
	}
	return event, nil
}

func (c *ZKECluster) getConfigFromDB(db storage.DB) (*zketypes.ZKEConfig, error) {
	state, err := c.getState(db)
	if err != nil {
		return nil, err
	}
	return state.CurrentState.ZKEConfig.DeepCopy(), nil
}

func (c *ZKECluster) getState(db storage.DB) (State, error) {
	s := State{
		FullState: &core.FullState{},
	}

	table, err := db.CreateOrGetTable(ZKEManagerTable)
	if err != nil {
		return State{}, fmt.Errorf("get table failed: %s", err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return State{}, fmt.Errorf("begin transaction failed: %s", err.Error())
	}

	defer tx.Commit()

	value, err := tx.Get(c.ClusterName)
	if value == nil {
		return s, nil
	}

	if err != nil {
		return State{}, fmt.Errorf("get cluster %s  state failed %s", c.ClusterName, err.Error())
	}

	if err := json.Unmarshal(value, &s); err != nil {
		return State{}, fmt.Errorf("unmarshal cluster %s state failed %s", c.ClusterName, err.Error())
	}
	s.DesiredState.CertificatesBundle = pki.TransformPEMToObject(s.DesiredState.CertificatesBundle)
	s.CurrentState.CertificatesBundle = pki.TransformPEMToObject(s.CurrentState.CertificatesBundle)

	return s, nil
}

func (c *ZKECluster) updateOrCreateState(db storage.DB, s State) error {
	table, err := db.CreateOrGetTable(ZKEManagerTable)
	if err != nil {
		return fmt.Errorf("get table failed %s", err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed %s", err.Error())
	}

	value, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal cluster %s state failed %s", c.ClusterName, err.Error())
	}

	defer tx.Rollback()

	stateJsonByte, err := tx.Get(c.ClusterName)
	if stateJsonByte != nil {
		if err := tx.Update(c.ClusterName, value); err != nil {
			return fmt.Errorf("update cluster %s  state failed %s", c.ClusterName, err.Error())
		}
	} else {
		if err := tx.Add(c.ClusterName, value); err != nil {
			return fmt.Errorf("add cluster %s  state failed %s", c.ClusterName, err.Error())
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cluster %s  state failed %s", c.ClusterName, err.Error())
	}

	return nil
}

func (c *ZKECluster) deleteState(db storage.DB) error {
	table, err := db.CreateOrGetTable(ZKEManagerTable)
	if err != nil {
		return fmt.Errorf("get table failed %s", err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed %s", err.Error())
	}

	defer tx.Rollback()
	if err := tx.Delete(c.ClusterName); err != nil {
		return fmt.Errorf("delete cluster %s  state failed %s", c.ClusterName, err.Error())
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cluster %s  state failed %s", c.ClusterName, err.Error())
	}

	return nil
}

func (c *ZKECluster) GetK8sClient(db storage.DB) (client.Client, *rest.Config, error) {
	state, err := c.getState(db)
	if err != nil {
		return nil, nil, err
	}

	k8sConfYaml := state.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConf, err := gok8sconfig.BuildConfig([]byte(k8sConfYaml))
	if err != nil {
		return nil, k8sConf, err
	}

	kubeClient, err := client.New(k8sConf, client.Options{})
	return kubeClient, k8sConf, err
}

func scClusterToZKECluster(cluster *types.Cluster) (*ZKECluster, error) {
	config := &zketypes.ZKEConfig{
		ClusterName:        cluster.Name,
		SingleCloudAddress: cluster.SingleCloudAddress,
	}
	config.Option.SSHUser = cluster.SSHUser
	config.Option.SSHPort = cluster.SSHPort
	config.Option.SSHKey = cluster.SSHKey
	config.Option.ClusterCidr = cluster.ClusterCidr
	config.Option.ServiceCidr = cluster.ServiceCidr
	config.Option.ClusterDomain = cluster.ClusterDomain
	config.Option.ClusterDNSServiceIP = cluster.ClusterDNSServiceIP
	config.Option.ClusterUpstreamDNS = cluster.ClusterUpstreamDNS
	config.Network.Plugin = cluster.Network.Plugin

	config.Nodes = []zketypes.ZKEConfigNode{}

	for _, node := range cluster.Nodes {
		n := zketypes.ZKEConfigNode{
			NodeName: node.NodeName,
			Address:  node.Address,
			Role:     node.Role,
		}
		config.Nodes = append(config.Nodes, n)
	}

	if cluster.PrivateRegistrys != nil {
		config.PrivateRegistries = []zketypes.PrivateRegistry{}
		for _, pr := range cluster.PrivateRegistrys {
			npr := zketypes.PrivateRegistry{
				User:     pr.User,
				Password: pr.Password,
				URL:      pr.URL,
				CAcert:   pr.CAcert,
			}
			config.PrivateRegistries = append(config.PrivateRegistries, npr)
		}
	}

	zc := &ZKECluster{
		ZKEConfig:  config,
		CreateTime: cluster.GetCreationTimestamp(),
	}

	if err := validateConfig(config); err != nil {
		return zc, err
	}

	return zc, nil
}
