package zke

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/hack/sockjs"
	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	gok8sconfig "github.com/zdnscloud/gok8s/client/config"
	zkecmd "github.com/zdnscloud/zke/cmd"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	zketypes "github.com/zdnscloud/zke/types"
)

const (
	ClusterRetryTimeInterval = time.Second * 10
)

type ZKECluster struct {
	*zketypes.ZKEConfig
	logCh      chan string
	logSession sockjs.Session
	cancel     context.CancelFunc
	lock       sync.Mutex
	CreateTime time.Time
}

func (c *ZKECluster) create(ctx context.Context, eventCh chan Event) {
	state := State{
		ZKEConfig:  c.ZKEConfig,
		CreateTime: c.CreateTime,
		FullState:  &core.FullState{},
	}

	event := Event{
		ClusterID: c.ZKEConfig.ClusterName,
		State:     state,
		Type:      CreateFailedEvent,
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
	// priintLog(c.logCh)

	event, err := upCluster(ctx, c.ZKEConfig, state, event, logger)
	if err != nil {
		log.Errorf("zke err info %s", err)
		eventCh <- event
		return
	}
	event.Type = CreateSuccessEvent
	eventCh <- event
}

func (c *ZKECluster) update(ctx context.Context, state State, eventCh chan Event) {
	event := Event{
		ClusterID: c.ZKEConfig.ClusterName,
		Type:      UpdateFailedEvent,
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
	// priintLog(c.logCh)

	event, err := upCluster(ctx, c.ZKEConfig, state, event, logger)
	if err != nil {
		log.Errorf("zke err info %s", err)
		eventCh <- event
		return
	}
	event.Type = UpdateSuccessEvent
	eventCh <- event
}

func upCluster(ctx context.Context, config *zketypes.ZKEConfig, state State, event Event, logger log.Logger) (Event, error) {
	newState, err := zkecmd.ClusterUpFromSingleCloud(ctx, config, state.FullState, logger)
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

	if err := deployZcloudProxy(kubeClient, config.ClusterName, config.SingleCloudAddress); err != nil {
		return event, err
	}

	state.FullState = newState
	state.IsUnvailable = false
	event.KubeClient = kubeClient
	event.K8sConfig = k8sConfig
	event.State = state
	return event, nil
}

func (c *ZKECluster) Cancel(eventCh chan Event) {
	c.cancel()
	eventCh <- Event{
		ClusterID: c.ClusterName,
		Type:      CancelSuccessEvent,
	}
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

	if cluster.PrivateRegistries != nil {
		config.PrivateRegistries = []zketypes.PrivateRegistry{}
		for _, pr := range cluster.PrivateRegistries {
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

func ZKEClusterToSCCluster(zc *ZKECluster) *types.Cluster {
	sc := &types.Cluster{}
	sc.Name = zc.ClusterName
	sc.SSHUser = zc.Option.SSHUser
	sc.SSHPort = zc.Option.SSHPort
	sc.SSHKey = zc.Option.SSHKey
	sc.ClusterCidr = zc.Option.ClusterCidr
	sc.ServiceCidr = zc.Option.ServiceCidr
	sc.ClusterDomain = zc.Option.ClusterDomain
	sc.ClusterDNSServiceIP = zc.Option.ClusterDNSServiceIP
	sc.ClusterUpstreamDNS = zc.Option.ClusterUpstreamDNS
	sc.SingleCloudAddress = zc.SingleCloudAddress

	sc.Network = types.ClusterNetwork{
		Plugin: zc.Network.Plugin,
	}

	sc.Nodes = []types.ClusterConfigNode{}
	for _, node := range zc.Nodes {
		n := types.ClusterConfigNode{
			NodeName: node.NodeName,
			Address:  node.Address,
			Role:     node.Role,
		}
		sc.Nodes = append(sc.Nodes, n)
	}

	if zc.PrivateRegistries != nil {
		sc.PrivateRegistries = []types.PrivateRegistry{}
		for _, pr := range zc.PrivateRegistries {
			npr := types.PrivateRegistry{
				User:     pr.User,
				Password: pr.Password,
				URL:      pr.URL,
				CAcert:   pr.CAcert,
			}
			sc.PrivateRegistries = append(sc.PrivateRegistries, npr)
		}
	}
	return sc
}

func (c *ZKECluster) initProcess(ctx context.Context, kubeConfig string, eventCh chan Event) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(ClusterRetryTimeInterval)
			k8sConfig, err := gok8sconfig.BuildConfig([]byte(kubeConfig))
			if err != nil {
				log.Errorf("build cluster %s k8sconfig err %s", c.ClusterName, err)
			}

			kubeClient, err := client.New(k8sConfig, client.Options{})
			if err == nil {
				eventCh <- Event{
					ClusterID:  c.ClusterName,
					K8sConfig:  k8sConfig,
					KubeClient: kubeClient,
					Type:       InitSuccessEvent,
				}
			}
		}
	}
}

func priintLog(logCh chan string) {
	go func() {
		for {
			l, ok := <-logCh
			if !ok {
				log.Infof("log channel has beed closed, will retrun")
				return
			}
			fmt.Printf(l)
		}
	}()
}
