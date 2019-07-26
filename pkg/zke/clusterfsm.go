package zke

import (
	"context"
	"sync"
	"time"

	"github.com/zdnscloud/singlecloud/hack/sockjs"
	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/fsm"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	zketypes "github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

const (
	InitEvent           = "init"
	InitSuccessEvent    = "initSuccess"
	InitFailedEvent     = "initFailed"
	CreateEvent         = "create"
	CreateSuccessEvent  = "createSuccess"
	CreateFailedEvent   = "createFailed"
	UpdateEvent         = "update"
	UpdateSuccessEvent  = "updateSuccess"
	UpdateFailedEvent   = "updateFailed"
	GetInfoFailedEvent  = "getInfoFailed"
	GetInfoSuccessEvent = "getInfoSuccess"
	CancelEvent         = "cancel"
	CancelSuccessEvent  = "cancelSuccess"
	DeleteEvent         = "delete"
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
	lock       sync.Mutex
	fsm        *fsm.FSM
}

type AddCluster struct {
	Cluster *Cluster
}

type DeleteCluster struct {
	Cluster *Cluster
}

type UpdateCluster struct {
	Cluster *Cluster
}

func newInitialCluster(name string) *Cluster {
	return newClusterWithStatus(name, types.CSInit)
}

func newClusterWithStatus(name string, status types.ClusterStatus) *Cluster {
	cluster := &Cluster{
		Name:   name,
		stopCh: make(chan struct{}),
	}

	fsm := fsm.NewFSM(
		string(status),
		fsm.Events{
			{Name: InitEvent, Src: []string{string(types.CSInit)}, Dst: string(types.CSConnecting)},
			{Name: InitSuccessEvent, Src: []string{string(types.CSConnecting)}, Dst: string(types.CSRunning)},
			{Name: InitFailedEvent, Src: []string{string(types.CSConnecting)}, Dst: string(types.CSUnavailable)},
			{Name: CreateEvent, Src: []string{string(types.CSInit)}, Dst: string(types.CSCreateing)},
			{Name: CreateSuccessEvent, Src: []string{string(types.CSCreateing)}, Dst: string(types.CSRunning)},
			{Name: CreateFailedEvent, Src: []string{string(types.CSCreateing)}, Dst: string(types.CSUnavailable)},
			{Name: UpdateEvent, Src: []string{string(types.CSRunning), string(types.CSUnavailable)}, Dst: string(types.CSUpdateing)},
			{Name: UpdateSuccessEvent, Src: []string{string(types.CSUpdateing)}, Dst: string(types.CSRunning)},
			{Name: UpdateFailedEvent, Src: []string{string(types.CSUpdateing)}, Dst: string(types.CSUnavailable)},
			{Name: GetInfoFailedEvent, Src: []string{string(types.CSRunning)}, Dst: string(types.CSUnreachable)},
			{Name: GetInfoSuccessEvent, Src: []string{string(types.CSUnreachable)}, Dst: string(types.CSRunning)},
			{Name: CancelEvent, Src: []string{string(types.CSUpdateing), string(types.CSCreateing), string(types.CSConnecting)}, Dst: string(types.CSCanceling)},
			{Name: CancelSuccessEvent, Src: []string{string(types.CSCanceling)}, Dst: string(types.CSUnavailable)},
			{Name: DeleteEvent, Src: []string{string(types.CSRunning), string(types.CSUnavailable), string(types.CSUnreachable)}, Dst: string(types.CSDestroy)},
		},
		fsm.Callbacks{
			InitSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				mgr.moveToreadyWithLock(cluster)
				mgr.sendPubEventWithLock(AddCluster{Cluster: cluster})
			},
			CreateSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				mgr.moveToreadyWithLock(cluster)
				mgr.updateClusterStateWithLock(cluster, state)
				mgr.sendPubEventWithLock(AddCluster{Cluster: cluster})
			},
			UpdateSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				mgr.moveToreadyWithLock(cluster)
				mgr.updateClusterStateWithLock(cluster, state)
				mgr.sendPubEventWithLock(UpdateCluster{Cluster: cluster})
			},
			CancelSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				mgr.setClusterUnavailable(cluster)
			},
		},
	)
	cluster.fsm = fsm
	return cluster
}
