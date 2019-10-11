package zke

import (
	"os"
	"testing"

	"github.com/zdnscloud/singlecloud/storage"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	fsmTestDbPath    = "fsm_tmp.db"
	fsmTestCluster   = "fsmTest"
	fsmTestScVersion = "v2.0.1"
)

func TestFsmInitSuccessEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSConnecting)
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(InitSuccessEvent, mgr)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", InitSuccessEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSRunning, "cluster status should be %s", types.CSRunning)
	})
}

func TestFsmInitFailedEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSConnecting)
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(InitFailedEvent, mgr)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", InitFailedEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSUnavailable, "cluster status should be %s", types.CSUnavailable)
	})
}

func TestFsmCreateSuccessEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSCreateing)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(CreateSuccessEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", CreateSuccessEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSRunning, "cluster status should be %s", types.CSRunning)
	})
}

func TestFsmCreateFailedEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSCreateing)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(CreateFailedEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", CreateFailedEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSUnavailable, "cluster status should be %s", types.CSUnavailable)
	})
}

func TestFsmUpdateSuccessEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSUpdateing)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(UpdateSuccessEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", CreateSuccessEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSRunning, "cluster status should be %s", types.CSRunning)
	})
}

func TestFsmUpdateFailedEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSUpdateing)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(UpdateFailedEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", UpdateFailedEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSUnavailable, "cluster status should be %s", types.CSUnavailable)
	})
}

func TestFsmGetInfoSuccessEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSUnreachable)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.clusters = append(mgr.clusters, cluster)
		err = cluster.fsm.Event(GetInfoSuccessEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", GetInfoSuccessEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSRunning, "cluster status should be %s", types.CSRunning)
	})
}

func TestFsmGetInfoFailedEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSRunning)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.clusters = append(mgr.clusters, cluster)
		err = cluster.fsm.Event(GetInfoFailedEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", GetInfoFailedEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSUnreachable, "cluster status should be %s", types.CSUnreachable)
	})
}

func TestFsmCancelEventWhenCreating(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSCreateing)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(CancelEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", CancelEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSCanceling, "cluster status should be %s", types.CSCanceling)
	})
}

func TestFsmCancelEventWhenUpdating(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSUpdateing)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(CancelEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", CancelEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSCanceling, "cluster status should be %s", types.CSCanceling)
	})
}

func TestFsmCancelEventWhenConnecting(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSConnecting)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(CancelEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", CancelEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSCanceling, "cluster status should be %s", types.CSCanceling)
	})
}

func TestFsmCancelSuccessEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSCanceling)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(CancelSuccessEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", CancelSuccessEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSUnavailable, "cluster status should be %s", types.CSUnavailable)
	})
}
