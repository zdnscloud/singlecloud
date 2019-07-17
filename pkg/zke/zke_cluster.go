package zke

import (
	"context"
	"sync"

	"github.com/zdnscloud/singlecloud/hack/sockjs"
	"github.com/zdnscloud/singlecloud/pkg/types"
	sctypes "github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	gok8sconfig "github.com/zdnscloud/gok8s/client/config"
	zkecmd "github.com/zdnscloud/zke/cmd"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	zketypes "github.com/zdnscloud/zke/types"
)

type ZKECluster struct {
	Config     *zketypes.ZKEConfig
	State      *core.FullState
	status     sctypes.ClusterStatus
	logCh      chan string
	logSession sockjs.Session
	Cancel     context.CancelFunc
	lock       sync.Mutex
}

func (zc *ZKECluster) AddNode(node *sctypes.Node, eventCh chan Event) error {
	config, err := getNewConfigForAddNode(zc.Config, node)
	if err != nil {
		return err
	}
	zc.Config = config

	ctx, cancel := context.WithCancel(context.Background())
	zc.Cancel = cancel

	go zc.update(ctx, eventCh)

	return nil
}

func (zc *ZKECluster) DeleteNode(node string, eventCh chan Event) error {
	config, err := getNewConfigForDeleteNode(zc.Config, node)
	if err != nil {
		return err
	}
	zc.Config = config

	ctx, cancel := context.WithCancel(context.Background())
	zc.Cancel = cancel

	go zc.update(ctx, eventCh)
	return nil
}

func (c *ZKECluster) create(ctx context.Context, eventCh chan Event) {
	var event = Event{
		ID:     c.Config.ClusterName,
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

	event, err := c.up(ctx, event, logger)
	if err != nil {
		log.Errorf("zke err info %s", err)
		eventCh <- event
	}
	event.Status = types.CSCreateSuccess
	eventCh <- event
}

func (c *ZKECluster) update(ctx context.Context, eventCh chan Event) {
	var event = Event{
		ID:     c.Config.ClusterName,
		Status: types.CSUpdateFailed,
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

	event, err := c.up(ctx, event, logger)
	if err != nil {
		log.Errorf("zke err info %s", err)
		eventCh <- event
	}
	event.Status = types.CSUpdateSuccess
	eventCh <- event
}

func (c *ZKECluster) up(ctx context.Context, event Event, l log.Logger) (Event, error) {
	state, err := zkecmd.ClusterUpFromRest(ctx, c.Config, c.State, l)
	if err != nil {
		return event, err
	}

	kubeConfigYaml := state.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConfig, err := gok8sconfig.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		return event, err
	}

	kubeClient, err := client.New(k8sConfig, client.Options{})
	if err != nil {
		return event, err
	}

	if err := deployZcloudProxy(kubeClient, c.Config.ClusterName, c.Config.SingleCloudAddress); err != nil {
		return event, err
	}

	event.KubeClient = kubeClient
	event.K8sConfig = k8sConfig
	event.State = state
	return event, nil
}
