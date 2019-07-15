package zke

import (
	"context"
	"sync"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/singlecloud/hack/sockjs"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

const (
	ClusterCreateFailed   = "CreateFailed"
	ClusterCreateComplete = "CreateComplete"
	ClusterCreateing      = "Createing"
	ClusterUpateing       = "updateing"
	ClusterUpateComplete  = "updated"
	ClusterUpateFailed    = "updateFailed"
)

type ZKE struct {
	Clusters map[string]*Cluster
	MsgCh    chan Msg
	Lock     sync.Mutex
}

type Cluster struct {
	Config     *types.ZKEConfig
	State      *core.FullState
	Status     string
	logCh      chan string
	logSession sockjs.Session
	CancelFunc context.CancelFunc
}

type Msg struct {
	ClusterName string
	KubeConfig  *rest.Config
	KubeClient  client.Client
	State       *core.FullState
	Status      string
	Error       error
}

func New() *ZKE {
	return &ZKE{
		Clusters: make(map[string]*Cluster),
		MsgCh:    make(chan Msg),
	}
}
