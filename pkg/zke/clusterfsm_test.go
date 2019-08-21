package zke

import (
	"fmt"
	"testing"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	ut "github.com/zdnscloud/cement/unittest"
)

const (
	testClusterName  = "local"
	testClusterName1 = "local1"
	testClusterName2 = "local2"
	testClusterName3 = "local3"
)

// event list: add->delete->add->delete->add->delete->add->delete->delete->delete->add
func pubEventLoop(eventCh chan interface{}, t *testing.T) {
	for {
		obj, ok := <-eventCh
		if !ok {
			return
		}
		switch obj.(type) {
		case AddCluster:
			fmt.Println("pubEventLoop receive add cluster event:")
		case DeleteCluster:
			fmt.Println("pubEventLoop receive delete cluster event")
		}
	}
}

func createZKEManagerObj(t *testing.T) (*ZKEManager, error) {
	db, err := storage.New("")
	if err != nil {
		return nil, err
	}
	return New(db)
}

func TestClusterFsm(t *testing.T) {
	mgr, err := createZKEManagerObj(t)
	if err != nil {
		fmt.Println("create zkeManager obj failed", err)
	}

	// cluster local:Init->Connecting->Running->Unreachable->Running->Updating->Unavailable->Updating->
	// Running->Updating->Canceling->Unavailable->Deleted
	cluster := newInitialCluster(testClusterName)
	mgr.addToUnreadyWithLock(cluster)

	state := clusterState{}
	createOrUpdateState(testClusterName, state, mgr.db)

	cluster.fsm.Event(InitEvent)
	ut.Equal(t, cluster.getStatus(), types.CSConnecting)

	cluster.fsm.Event(InitSuccessEvent, mgr) //there will send an add cluster event
	ut.Equal(t, cluster.getStatus(), types.CSRunning)

	cluster.fsm.Event(GetInfoFailedEvent)
	ut.Equal(t, cluster.getStatus(), types.CSUnreachable)

	cluster.fsm.Event(GetInfoSuccessEvent)
	ut.Equal(t, cluster.getStatus(), types.CSRunning)

	mgr.moveTounready(cluster)
	cluster.fsm.Event(UpdateEvent)
	ut.Equal(t, cluster.getStatus(), types.CSUpdateing)

	cluster.fsm.Event(UpdateFailedEvent)
	ut.Equal(t, cluster.getStatus(), types.CSUnavailable)

	cluster.fsm.Event(UpdateEvent)
	ut.Equal(t, cluster.getStatus(), types.CSUpdateing)

	cluster.fsm.Event(UpdateSuccessEvent, mgr, state) //there will send a update cluster event
	ut.Equal(t, cluster.getStatus(), types.CSRunning)

	mgr.moveTounready(cluster)
	cluster.fsm.Event(UpdateEvent)
	ut.Equal(t, cluster.getStatus(), types.CSUpdateing)
	cluster.fsm.Event(CancelEvent)
	ut.Equal(t, cluster.getStatus(), types.CSCanceling)

	cluster.fsm.Event(CancelSuccessEvent, mgr)
	ut.Equal(t, cluster.getStatus(), types.CSUnavailable)
	mgr.Delete(testClusterName) //there will send a delete cluster event

	// cluster local1:Init->Creating->Running->Updating->Unavailable->Updating->Running->Unreachable->Deleted
	cluster1 := newInitialCluster(testClusterName1)
	createOrUpdateState(testClusterName1, state, mgr.db)
	mgr.addToUnreadyWithLock(cluster1)

	cluster1.fsm.Event(CreateEvent)
	ut.Equal(t, cluster1.getStatus(), types.CSCreateing)

	cluster1.fsm.Event(CreateSuccessEvent, mgr, state) //there will send an add cluster event
	ut.Equal(t, cluster1.getStatus(), types.CSRunning)

	mgr.moveTounready(cluster)
	cluster1.fsm.Event(UpdateEvent)
	ut.Equal(t, cluster1.getStatus(), types.CSUpdateing)

	cluster1.fsm.Event(UpdateFailedEvent)
	ut.Equal(t, cluster1.getStatus(), types.CSUnavailable)

	cluster1.fsm.Event(UpdateEvent)
	ut.Equal(t, cluster1.getStatus(), types.CSUpdateing)

	cluster1.fsm.Event(UpdateSuccessEvent, mgr, state) //there will send an update cluster event
	ut.Equal(t, cluster1.getStatus(), types.CSRunning)

	cluster1.fsm.Event(GetInfoFailedEvent)
	ut.Equal(t, cluster1.getStatus(), types.CSUnreachable)
	mgr.Delete(testClusterName1) //there will send a delete cluster event

	// cluster local2:Init->Creating->Unavailable->Deleted
	cluster2 := newInitialCluster(testClusterName2)
	createOrUpdateState(testClusterName2, state, mgr.db)
	mgr.addToUnreadyWithLock(cluster2)

	cluster2.fsm.Event(CreateEvent)
	ut.Equal(t, cluster2.getStatus(), types.CSCreateing)

	cluster2.fsm.Event(CreateFailedEvent)
	ut.Equal(t, cluster2.getStatus(), types.CSUnavailable)
	mgr.Delete(testClusterName2) //there will send a delete cluster event

	// cluster local3:Init->Creating->Canceling->Unavailable->Updating->Running
	cluster3 := newInitialCluster(testClusterName3)
	mgr.addToUnreadyWithLock(cluster3)

	cluster3.fsm.Event(CreateEvent)
	createOrUpdateState(testClusterName3, state, mgr.db)
	ut.Equal(t, cluster3.getStatus(), types.CSCreateing)

	cluster3.fsm.Event(CancelEvent)
	ut.Equal(t, cluster3.getStatus(), types.CSCanceling)
	cluster3.fsm.Event(CancelSuccessEvent, mgr)
	ut.Equal(t, cluster3.getStatus(), types.CSUnavailable)

	cluster3.fsm.Event(UpdateEvent)
	ut.Equal(t, cluster3.getStatus(), types.CSUpdateing)

	cluster3.fsm.Event(UpdateSuccessEvent, mgr, state) //there will send an update cluster event
	ut.Equal(t, cluster3.getStatus(), types.CSRunning)
	go pubEventLoop(mgr.PubEventCh, t)
	time.Sleep(time.Second * 5)
	close(mgr.PubEventCh)
}
