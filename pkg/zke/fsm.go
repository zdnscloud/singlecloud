package zke

import (
	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/fsm"
)

const (
	InitSuccessEvent    = "initSuccess"
	InitFailedEvent     = "initFailed"
	CreateSuccessEvent  = "createSuccess"
	CreateFailedEvent   = "createFailed"
	UpdateEvent         = "update"
	UpdateSuccessEvent  = "updateSuccess"
	UpdateFailedEvent   = "updateFailed"
	GetInfoFailedEvent  = "getInfoFailed"
	GetInfoSuccessEvent = "getInfoSuccess"
	CancelEvent         = "cancel"
	CancelSuccessEvent  = "cancelSuccess"
)

func newClusterFsm(cluster *Cluster, initialStatus types.ClusterStatus) *fsm.FSM {
	return fsm.NewFSM(
		string(initialStatus),
		fsm.Events{
			{Name: InitSuccessEvent, Src: []string{string(types.CSConnecting)}, Dst: string(types.CSRunning)},
			{Name: InitFailedEvent, Src: []string{string(types.CSConnecting)}, Dst: string(types.CSUnavailable)},
			{Name: CreateSuccessEvent, Src: []string{string(types.CSCreateing)}, Dst: string(types.CSRunning)},
			{Name: CreateFailedEvent, Src: []string{string(types.CSCreateing)}, Dst: string(types.CSUnavailable)},
			{Name: UpdateEvent, Src: []string{string(types.CSRunning), string(types.CSUnavailable)}, Dst: string(types.CSUpdateing)},
			{Name: UpdateSuccessEvent, Src: []string{string(types.CSUpdateing)}, Dst: string(types.CSRunning)},
			{Name: UpdateFailedEvent, Src: []string{string(types.CSUpdateing)}, Dst: string(types.CSUnavailable)},
			{Name: GetInfoFailedEvent, Src: []string{string(types.CSRunning)}, Dst: string(types.CSUnreachable)},
			{Name: GetInfoSuccessEvent, Src: []string{string(types.CSUnreachable)}, Dst: string(types.CSRunning)},
			{Name: CancelEvent, Src: []string{string(types.CSUpdateing), string(types.CSCreateing), string(types.CSConnecting)}, Dst: string(types.CSCanceling)},
			{Name: CancelSuccessEvent, Src: []string{string(types.CSCanceling)}, Dst: string(types.CSUnavailable)},
		},
		fsm.Callbacks{
			InitSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				mgr.MoveToReady(cluster)
				mgr.SendEvent(AddCluster{Cluster: cluster})
			},
			CreateSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				mgr.MoveToReady(cluster)
				createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDB())
				mgr.SendEvent(AddCluster{Cluster: cluster})
			},
			CreateFailedEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDB())
			},
			UpdateSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				cluster.stopCh = make(chan struct{})
				cluster.setCache(cluster.K8sConfig)
				mgr.MoveToReady(cluster)
				createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDB())
				mgr.SendEvent(AddCluster{Cluster: cluster})
			},
			CancelSuccessEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				cluster.isCanceled = false
				createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDB())
				cluster.setUnavailable(mgr.GetDB())
			},
		},
	)
}
