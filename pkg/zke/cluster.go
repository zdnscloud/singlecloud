package zke

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/hack/sockjs"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/cement/fsm"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	zketypes "github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

const (
	ClusterRetryInterval = time.Second * 10
)

type Cluster struct {
	Name       string
	CreateTime time.Time
	KubeClient client.Client
	Cache      cache.Cache
	K8sConfig  *rest.Config
	stopCh     chan struct{}
	config     *zketypes.ZKEConfig
	logCh      chan string
	logSession sockjs.Session
	cancel     context.CancelFunc
	isCanceled bool
	lock       sync.Mutex
	fsm        *fsm.FSM
	scVersion  string
}

type AddCluster struct {
	Cluster *Cluster
}

type DeleteCluster struct {
	Cluster *Cluster
}

func newCluster(name string, initialStatus types.ClusterStatus) *Cluster {
	cluster := &Cluster{
		Name:   name,
		stopCh: make(chan struct{}),
	}

	fsm := newClusterFsm(cluster, initialStatus)
	cluster.fsm = fsm
	return cluster
}

func (c *Cluster) IsReady() bool {
	status := c.getStatus()
	if status == types.CSRunning || status == types.CSUnreachable {
		return true
	}
	return false
}

func (c *Cluster) ToTypesCluster() *types.Cluster {
	cluster := c.toTypesCluster()
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

func (c *Cluster) GetNodeIpsByRole(role types.NodeRole) []string {
	ips := []string{}
	cluster := c.toTypesCluster()
	for _, n := range cluster.Nodes {
		if n.HasRole(role) {
			ips = append(ips, n.Address)
		}
	}
	return ips
}

func (c *Cluster) Cancel() error {
	if err := c.fsm.Event(CancelEvent); err != nil {
		return err
	}
	c.cancel()
	c.isCanceled = true
	return nil
}

func (c *Cluster) CanUpdate() bool {
	return c.fsm.Can(UpdateEvent)
}

func (c *Cluster) getStatus() types.ClusterStatus {
	return types.ClusterStatus(c.fsm.Current())
}

func (c *Cluster) InitLoop(ctx context.Context, kubeConfig string, mgr *ZKEManager, state clusterState) {
	k8sConfig, err := config.BuildConfig([]byte(kubeConfig))
	if err != nil {
		log.Errorf("build cluster %s k8sconfig failed %s", c.Name, err)
		if err := c.fsm.Event(InitFailedEvent); err != nil {
			log.Warnf("send cluster fsm %s event failed %s", InitFailedEvent, err.Error())
		}
		return
	}

	for {
		if c.isCanceled {
			if err := c.fsm.Event(CancelSuccessEvent, mgr, state); err != nil {
				log.Warnf("send cluster fsm %s event failed %s", CancelSuccessEvent, err.Error())
			}
			return
		}

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
					if err := c.fsm.Event(InitFailedEvent); err != nil {
						log.Warnf("send cluster fsm %s event failed %s", InitFailedEvent, err.Error())
					}
					return
				}
				if err := c.fsm.Event(InitSuccessEvent, mgr); err != nil {
					log.Warnf("send cluster fsm %s event failed %s", InitSuccessEvent, err.Error())
				}
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

func (c *Cluster) setUnavailable(db storage.DB) error {
	state, err := getClusterFromDB(c.Name, db)
	if err != nil {
		return err
	}
	// only connecting status cancel action need update and write db
	if state.IsUnvailable {
		return nil
	}
	state.IsUnvailable = true
	return createOrUpdateClusterFromDB(c.Name, state, db)
}

func (c *Cluster) Create(ctx context.Context, state clusterState, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			if err := c.fsm.Event(CreateFailedEvent, mgr, state); err != nil {
				log.Warnf("send cluster fsm %s event failed %s", CreateFailedEvent, err.Error())
			}
		}
	}()

	logger, logCh := log.NewLog4jBufLogger(MaxZKELogLines, log.Info)
	defer logger.Close()
	c.logCh = logCh

	zkeState, k8sConfig, kubeClient, err := upZKECluster(ctx, c.config, state.FullState, logger)
	state.FullState = zkeState
	if c.isCanceled {
		if err := c.fsm.Event(CancelSuccessEvent, mgr, state); err != nil {
			log.Warnf("send cluster fsm %s event failed %s", CancelSuccessEvent, err.Error())
		}
		return
	}
	if err != nil {
		log.Errorf("zke err info %s", err)
		logger.Error(err.Error())
		if err := c.fsm.Event(CreateFailedEvent, mgr, state); err != nil {
			log.Warnf("send cluster fsm %s event failed %s", CreateFailedEvent, err.Error())
		}
		return
	}

	c.KubeClient = kubeClient
	c.K8sConfig = k8sConfig

	state.FullState = zkeState
	state.IsUnvailable = false

	if err := c.setCache(k8sConfig); err != nil {
		if err := c.fsm.Event(CreateFailedEvent, mgr, state); err != nil {
			log.Warnf("send cluster fsm %s event failed %s", CreateFailedEvent, err.Error())
		}
		return
	}

	if err := c.fsm.Event(CreateSuccessEvent, mgr, state); err != nil {
		log.Warnf("send cluster fsm %s event failed %s", CreateSuccessEvent, err.Error())
	}
}

func (c *Cluster) Update(ctx context.Context, state clusterState, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			if err := c.fsm.Event(UpdateFailedEvent, mgr, state); err != nil {
				log.Warnf("send cluster fsm %s event failed %s", UpdateFailedEvent, err.Error())
			}
		}
	}()

	logger, logCh := log.NewLog4jBufLogger(MaxZKELogLines, log.Info)
	defer logger.Close()
	c.logCh = logCh

	zkeState, k8sConfig, kubeClient, err := upZKECluster(ctx, c.config, state.FullState, logger)
	state.FullState = zkeState
	if c.isCanceled {
		if err := c.fsm.Event(CancelSuccessEvent, mgr, state); err != nil {
			log.Warnf("send cluster fsm %s event failed %s", CancelSuccessEvent, err.Error())
		}
		return
	}
	if err != nil {
		log.Errorf("zke err info %s", err)
		logger.Error(err.Error())
		if err := c.fsm.Event(UpdateFailedEvent, mgr, state); err != nil {
			log.Warnf("send cluster fsm %s event failed %s", UpdateFailedEvent, err.Error())
		}
		return
	}

	if err := c.setCache(k8sConfig); err != nil {
		if err := c.fsm.Event(UpdateFailedEvent, mgr, state); err != nil {
			log.Warnf("send cluster fsm %s event failed %s", UpdateFailedEvent, err.Error())
		}
		return
	}

	c.stopCh = make(chan struct{})
	c.KubeClient = kubeClient
	state.FullState = zkeState
	state.IsUnvailable = false

	if err := c.fsm.Event(UpdateSuccessEvent, mgr, state); err != nil {
		log.Warnf("send cluster fsm %s event failed %s", UpdateSuccessEvent, err.Error())
	}
}

func (c *Cluster) Destroy(ctx context.Context, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			if err := c.fsm.Event(DeleteSuccessEvent, mgr); err != nil {
				log.Warnf("send cluster fsm %s event failed %s", DeleteSuccessEvent, err.Error())
			}
		}
	}()

	logger, logCh := log.NewLog4jBufLogger(MaxZKELogLines, log.Info)
	defer logger.Close()
	c.logCh = logCh

	if err := removeZKECluster(ctx, c.config, logger); err != nil {
		log.Errorf("zke err info %s", err)
		logger.Error(err.Error())
	}

	if err := c.fsm.Event(DeleteSuccessEvent, mgr); err != nil {
		log.Warnf("send cluster fsm %s event failed %s", DeleteSuccessEvent, err.Error())
	}
	return
}

func (c *Cluster) toTypesCluster() *types.Cluster {
	sc := &types.Cluster{}
	sc.Name = c.Name
	sc.SSHUser = c.config.Option.SSHUser
	sc.SSHPort = c.config.Option.SSHPort
	sc.ClusterCidr = c.config.Option.ClusterCidr
	sc.ServiceCidr = c.config.Option.ServiceCidr
	sc.ClusterDomain = c.config.Option.ClusterDomain
	sc.ClusterDNSServiceIP = c.config.Option.ClusterDNSServiceIP
	sc.ClusterUpstreamDNS = c.config.Option.ClusterUpstreamDNS
	sc.SingleCloudAddress = c.config.SingleCloudAddress
	sc.ScVersion = c.scVersion

	sc.Network = types.ClusterNetwork{
		Plugin: c.config.Network.Plugin,
	}

	for _, node := range c.config.Nodes {
		n := types.Node{
			Name:    node.NodeName,
			Address: node.Address,
		}
		for _, role := range node.Role {
			n.Roles = append(n.Roles, types.NodeRole(role))
		}
		sc.Nodes = append(sc.Nodes, n)
	}
	sc.NodesCount = len(sc.Nodes)

	if c.config.PrivateRegistries != nil {
		sc.PrivateRegistries = []types.PrivateRegistry{}
		for _, pr := range c.config.PrivateRegistries {
			npr := types.PrivateRegistry{
				User:     pr.User,
				Password: pr.Password,
				URL:      pr.URL,
				CAcert:   pr.CAcert,
			}
			sc.PrivateRegistries = append(sc.PrivateRegistries, npr)
		}
	}

	sc.SetID(c.Name)
	sc.SetCreationTimestamp(c.CreateTime)
	sc.Status = c.getStatus()
	return sc
}

func (c *Cluster) GetKubeConfig(user string, db storage.DB) (string, error) {
	state, err := getClusterFromDB(c.Name, db)
	if err != nil {
		return "", err
	}
	if state.FullState.CurrentState.CertificatesBundle != nil {
		kubeConfigCert, ok := state.CurrentState.CertificatesBundle[user]
		if !ok {
			return "", fmt.Errorf("cluster %s user %s cert doesn't exist", c.Name, user)
		}
		return kubeConfigCert.Config, nil
	}
	return "", nil
}
