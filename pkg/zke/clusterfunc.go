package zke

import (
	"context"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"k8s.io/client-go/rest"
)

const (
	ClusterRetryInterval = time.Second * 10
)

func (c *Cluster) ToTypesCluster() *types.Cluster {
	cluster := zkeClusterToSCCluster(c)
	// unready cluster do not get cluster version info
	if !c.IsReady() {
		return cluster
	}
	if c.KubeClient == nil {
		return cluster
	}
	version, err := c.KubeClient.ServerVersion()
	if err != nil {
		c.fsm.Event(GetInfoFailedEvent)
		return cluster
	}
	c.fsm.Event(GetInfoSuccessEvent)
	cluster.Version = version.GitVersion
	return cluster
}

func (c *Cluster) IsReady() bool {
	status := c.getStatus()
	if status == types.CSUnavailable || status == types.CSConnecting || status == types.CSCreateing || status == types.CSUpdateing || status == types.CSCanceling {
		return false
	}
	return true
}

func (c *Cluster) getStatus() types.ClusterStatus {
	return types.ClusterStatus(c.fsm.Current())
}

func (c *Cluster) initLoop(ctx context.Context, kubeConfig string, mgr *ZKEManager) {
	k8sConfig, err := config.BuildConfig([]byte(kubeConfig))
	if err != nil {
		log.Errorf("build cluster %s k8sconfig failed %s", c.Name, err)
		c.fsm.Event(InitFailedEvent)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var options client.Options
			options.Scheme = client.GetDefaultScheme()
			storagev1.AddToScheme(options.Scheme)
			kubeClient, err := client.New(k8sConfig, options)
			if err == nil {
				c.KubeClient = kubeClient
				if err := c.setCache(k8sConfig); err != nil {
					log.Errorf("set cluster %s cache failed %s", c.Name, err)
					c.fsm.Event(InitFailedEvent)
					return
				}
				c.fsm.Event(InitSuccessEvent, mgr)
				return
			}
			time.Sleep(ClusterRetryInterval)
		}
	}
}

func (c *Cluster) setCache(k8sConfig *rest.Config) error {
	c.K8sConfig = k8sConfig
	cache, err := cache.New(k8sConfig, cache.Options{})
	if err != nil {
		return err
	}
	go cache.Start(c.stopCh)
	cache.WaitForCacheSync(c.stopCh)
	c.Cache = cache
	return nil
}

func (c *Cluster) create(ctx context.Context, state clusterState, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			c.fsm.Event(CreateFailedEvent, mgr, state)
		}
	}()

	logger, logCh := log.NewLog4jBufLogger(MaxZKELogLines, log.Info)
	defer logger.Close()
	c.logCh = logCh

	zkeState, k8sConfig, kubeClient, err := upCluster(ctx, c.config, state.FullState, logger, true)
	state.FullState = zkeState
	if c.isCanceled {
		c.fsm.Event(CancelSuccessEvent, mgr, state)
		return
	}
	if err != nil {
		log.Errorf("zke err info %s", err)
		c.fsm.Event(CreateFailedEvent, mgr, state)
		return
	}

	c.KubeClient = kubeClient
	c.K8sConfig = k8sConfig

	state.FullState = zkeState
	state.IsUnvailable = false

	if err := c.setCache(k8sConfig); err != nil {
		c.fsm.Event(CreateFailedEvent, mgr, state)
		return
	}

	c.fsm.Event(CreateSuccessEvent, mgr, state)
}

func (c *Cluster) update(ctx context.Context, state clusterState, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			c.fsm.Event(UpdateFailedEvent, mgr, state)
		}
	}()

	logger, logCh := log.NewLog4jBufLogger(MaxZKELogLines, log.Info)
	defer logger.Close()
	c.logCh = logCh

	zkeState, k8sConfig, kubeClient, err := upCluster(ctx, c.config, state.FullState, logger, false)
	state.FullState = zkeState
	if c.isCanceled {
		c.fsm.Event(CancelSuccessEvent, mgr, state)
		return
	}
	if err != nil {
		log.Errorf("zke err info %s", err)
		c.fsm.Event(UpdateFailedEvent, mgr, state)
		return
	}

	c.K8sConfig = k8sConfig
	c.KubeClient = kubeClient

	state.FullState = zkeState
	state.IsUnvailable = false

	c.fsm.Event(UpdateSuccessEvent, mgr, state)
}
