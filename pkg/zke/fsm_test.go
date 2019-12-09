package zke

import (
	"os"
	"testing"

	"github.com/zdnscloud/singlecloud/pkg/types"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/kvzoo/backend/bolt"
)

const (
	fsmTestDbPath    = "fsm_tmp.db"
	fsmTestCluster   = "fsmTest"
	fsmTestScVersion = "v2.0.1"
)

func TestFsmCreateSuccessEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSCreating)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		cluster.event(CreateSucceedEvent, mgr, state)
		ut.Assert(t, cluster.getStatus() == types.CSRunning, "cluster status should be %s", types.CSRunning)
	})
}

func TestFsmCreateFailedEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSCreating)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		err = cluster.fsm.Event(CreateFailedEvent, mgr, state)
		ut.Assert(t, err == nil, "send %s fsm event should succeed: %s", CreateFailedEvent, err)
		ut.Assert(t, cluster.getStatus() == types.CSCreateFailed, "cluster status should be %s", types.CSCreateFailed)
	})
}

func TestFsmCreateCanceledEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSCreating)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		cluster.event(CreateCanceledEvent, mgr, state)
		ut.Assert(t, cluster.getStatus() == types.CSCreateFailed, "cluster status should be %s", types.CSCreateFailed)
	})
}

func TestFsmContinueCreateEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSCreateFailed)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		cluster.event(ContinuteCreateEvent, mgr, state)
		ut.Assert(t, cluster.getStatus() == types.CSCreating, "cluster status should be %s", types.CSCreating)
	})
}

func TestFsmUpdateCompletedEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSUpdating)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		cluster.event(UpdateCompletedEvent, mgr, state)
		ut.Assert(t, cluster.getStatus() == types.CSRunning, "cluster status should be %s", types.CSRunning)
	})
}

func TestFsmUpdateCanceledEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSUpdating)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.add(cluster)
		cluster.event(UpdateCanceledEvent, mgr, state)
		ut.Assert(t, cluster.getStatus() == types.CSRunning, "cluster status should be %s", types.CSRunning)
	})
}

func TestFsmGetInfoSuccessEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSUnreachable)
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.clusters = append(mgr.clusters, cluster)
		cluster.Event(GetInfoSucceedEvent)
		ut.Assert(t, cluster.getStatus() == types.CSRunning, "cluster status should be %s", types.CSRunning)
	})
}

func TestFsmGetInfoFailedEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSRunning)
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.clusters = append(mgr.clusters, cluster)
		cluster.Event(GetInfoFailedEvent)
		ut.Assert(t, cluster.getStatus() == types.CSUnreachable, "cluster status should be %s", types.CSUnreachable)
	})
}

func TestFsmDeleteEventWhenRunning(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSRunning)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.clusters = append(mgr.clusters, cluster)
		cluster.event(DeleteEvent, mgr, state)
		ut.Assert(t, cluster.getStatus() == types.CSDeleting, "cluster status should be %s", types.CSDeleting)
	})
}

func TestFsmDeleteEventWhenUnreached(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSUnreachable)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.clusters = append(mgr.clusters, cluster)
		cluster.event(DeleteEvent, mgr, state)
		ut.Assert(t, cluster.getStatus() == types.CSDeleting, "cluster status should be %s", types.CSDeleting)
	})
}

func TestFsmDeleteEventWhenCreateFailed(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSCreateFailed)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.clusters = append(mgr.clusters, cluster)
		cluster.event(DeleteEvent, mgr, state)
		ut.Assert(t, cluster.getStatus() == types.CSDeleting, "cluster status should be %s", types.CSDeleting)
	})
}

func TestFsmDeleteCompleteEvent(t *testing.T) {
	cluster := newCluster(fsmTestCluster, types.CSDeleting)
	state := clusterState{}
	ut.WithTempFile(t, fsmTestDbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		mgr, err := New(db, fsmTestScVersion, nil)
		ut.Assert(t, err == nil, "create zke manager obj should succeed: %s", err)
		mgr.clusters = append(mgr.clusters, cluster)
		cluster.event(DeleteCompletedEvent, mgr, state)
		ut.Assert(t, cluster.getStatus() == types.CSDeleted, "cluster status should be %s", types.CSDeleted)
	})
}
