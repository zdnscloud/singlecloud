package zke

import (
	"context"
	"sync"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/singlecloud/hack/sockjs"
	sctypes "github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

type ZKEManager map[string]*ZKECluster

type ZKECluster struct {
	Config     *types.ZKEConfig
	State      *core.FullState
	logCh      chan string
	logSession sockjs.Session
	Cancel     context.CancelFunc
	lock       sync.Mutex
}

type Event struct {
	ID         string
	Status     sctypes.ClusterStatus
	State      *core.FullState
	KubeClient client.Client
	K8sConfig  *rest.Config
}

func New() ZKEManager {
	return map[string]*ZKECluster{}
}
