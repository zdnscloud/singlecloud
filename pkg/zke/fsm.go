package zke

import (
	"fmt"

	"github.com/zdnscloud/singlecloud/pkg/alarm"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
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

	DeleteFailedEvent = "deleteFailed"
	UpdateFailedEvent = "updateFailed"

	clusterKindName = "cluster"
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
				mgr, state, _, err := getFsmEventArgs(e)
				if err != nil {
					log.Warnf("fsm %s callback failed %s", CreateSucceedEvent, err.Error())
				}

				if err := mgr.Logger.Delete(cluster.Name); err != nil {
					log.Warnf("fsm %s callback failed %s", CreateSucceedEvent, err.Error())
				}

				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
				eventbus.PublishResourceCreateEvent(cluster.ToScCluster())
			},
			CreateFailedEvent: func(e *fsm.Event) {
				mgr, state, errMsg, err := getFsmEventArgs(e)
				if err != nil {
					log.Warnf("fsm %s callback failed %s", CreateFailedEvent, err.Error())
					return
				}

				if errMsg != "" {
					alarm.New().Kind(clusterKindName).Cluster(cluster.Name).Name(cluster.Name).Reason(CreateFailedEvent).Message(errMsg).Publish()
				}

				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
			CreateCanceledEvent: func(e *fsm.Event) {
				mgr, state, _, err := getFsmEventArgs(e)
				if err != nil {
					log.Warnf("fsm %s callback failed %s", CreateCanceledEvent, err.Error())
					return
				}

				cluster.isCanceled = false
				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
			UpdateCompletedEvent: func(e *fsm.Event) {
				mgr, state, errMsg, err := getFsmEventArgs(e)
				if err != nil {
					log.Warnf("fsm %s callback failed %s", UpdateCompletedEvent, err.Error())
					return
				}

				if err := mgr.Logger.Delete(cluster.Name); err != nil {
					log.Warnf("fsm %s callback failed %s", UpdateCompletedEvent, err.Error())
				}

				if errMsg != "" {
					alarm.New().Kind(clusterKindName).Cluster(cluster.Name).Name(cluster.Name).Reason(UpdateFailedEvent).Message(errMsg).Publish()
				}

				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
			UpdateCanceledEvent: func(e *fsm.Event) {
				mgr, state, _, err := getFsmEventArgs(e)
				if err != nil {
					log.Warnf("fsm %s callback failed %s", UpdateCompletedEvent, err.Error())
					return
				}

				cluster.isCanceled = false
				if err := createOrUpdateClusterFromDB(cluster.Name, state, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
			DeleteCompletedEvent: func(e *fsm.Event) {
				mgr, _, errMsg, err := getFsmEventArgs(e)
				if err != nil {
					log.Warnf("fsm %s callback failed %s", DeleteCompletedEvent, err.Error())
					return
				}

				if errMsg != "" {
					alarm.New().Kind(clusterKindName).Cluster(cluster.Name).Name(cluster.Name).Reason(DeleteFailedEvent).Message(errMsg).Publish()
				}

				mgr.Remove(cluster)
				if err := deleteClusterFromDB(cluster.Name, mgr.GetDBTable()); err != nil {
					log.Warnf("update db failed after cluster %s %s event %s", cluster.Name, e.Event, err.Error())
				}
			},
		},
	)
}

func getFsmEventArgs(e *fsm.Event) (*ZKEManager, clusterState, string, error) {
	zkeMgr, ok := e.Args[0].(*ZKEManager)
	if !ok {
		return nil, clusterState{}, "", fmt.Errorf("get zke manager from fsm event failed")
	}

	state, ok := e.Args[1].(clusterState)
	if !ok {
		return nil, clusterState{}, "", fmt.Errorf("get clusterState from fsm event failed")
	}

	errMessage, ok := e.Args[2].(string)
	if !ok {
		return nil, clusterState{}, "", fmt.Errorf("get errMessage from fsm event failed")
	}
	return zkeMgr, state, errMessage, nil
}
