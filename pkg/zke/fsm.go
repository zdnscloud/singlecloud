package zke

import (
	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/fsm"
	"github.com/zdnscloud/cement/log"
)

const (
	CreateSucceedEvent   = "createSucceed"
	CreateFailedEvent    = "createFailed"
	CreateCanceledEvent  = "createCanceled"
	ContinuteCreateEvent = "continuteCreate"
	UpdateEvent          = "update"
	UpdateCompletedEvent = "updateCompleted"
	UpdateCanceledEvent  = "updateCanceled"
	GetInfoFailedEvent   = "getInfoFailed"
	GetInfoSucceedEvent  = "getInfoSucceed"
	DeleteEvent          = "delete"
	DeleteCompletedEvent = "deleteCompleted"
)

func newClusterFsm(cluster *Cluster, initialStatus types.ClusterStatus) *fsm.FSM {
	return fsm.NewFSM(
		string(initialStatus),
		fsm.Events{
			{Name: CreateSucceedEvent, Src: []string{string(types.CSCreating)}, Dst: string(types.CSRunning)},
			{Name: CreateFailedEvent, Src: []string{string(types.CSCreating)}, Dst: string(types.CSCreateFailed)},
			{Name: CreateCanceledEvent, Src: []string{string(types.CSCreating)}, Dst: string(types.CSCreateFailed)},
			{Name: ContinuteCreateEvent, Src: []string{string(types.CSCreateFailed)}, Dst: string(types.CSCreating)},
			{Name: UpdateEvent, Src: []string{string(types.CSRunning)}, Dst: string(types.CSUpdating)},
			{Name: UpdateCompletedEvent, Src: []string{string(types.CSUpdating)}, Dst: string(types.CSRunning)},
			{Name: UpdateCanceledEvent, Src: []string{string(types.CSUpdating)}, Dst: string(types.CSRunning)},
			{Name: GetInfoFailedEvent, Src: []string{string(types.CSRunning)}, Dst: string(types.CSUnreachable)},
			{Name: GetInfoSucceedEvent, Src: []string{string(types.CSUnreachable)}, Dst: string(types.CSRunning)},
			{Name: DeleteEvent, Src: []string{string(types.CSRunning), string(types.CSCreateFailed), string(types.CSUnreachable)}, Dst: string(types.CSDeleting)},
			{Name: DeleteCompletedEvent, Src: []string{string(types.CSDeleting)}, Dst: string(types.CSDeleted)},
		},
		fsm.Callbacks{
			CreateSucceedEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				cluster.logCh = nil
				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
				mgr.SendEvent(AddCluster{Cluster: cluster})
			},
			CreateFailedEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
			CreateCanceledEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				cluster.isCanceled = false
				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
			UpdateCompletedEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				cluster.logCh = nil
				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
			UpdateCanceledEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				state := e.Args[1].(clusterState)
				cluster.isCanceled = false
				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
			DeleteCompletedEvent: func(e *fsm.Event) {
				mgr := e.Args[0].(*ZKEManager)
				mgr.Remove(cluster)
				if err := deleteClusterFromDB(cluster.Name, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
		},
	)
}
