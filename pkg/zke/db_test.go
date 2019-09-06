package zke

import (
	"os"
	"testing"
	"time"

	"github.com/zdnscloud/singlecloud/storage"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/zke/core"
	zketypes "github.com/zdnscloud/zke/types"
)

func TestClusterDB(t *testing.T) {
	dbPath := "cluster.db"
	clusterName := "local"
	ut.WithTempFile(t, dbPath, func(t *testing.T, f *os.File) {
		db, err := storage.CreateWithPath(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)

		newClusterState := clusterState{
			FullState:    &core.FullState{},
			ZKEConfig:    &zketypes.ZKEConfig{},
			CreateTime:   time.Now(),
			IsUnvailable: false,
			ScVersion:    "v1.0",
		}

		err = createOrUpdateClusterFromDB(clusterName, newClusterState, db)
		ut.Assert(t, err == nil, "create cluster from db should succeed: %s", err)

		newClusterState.IsUnvailable = true
		err = createOrUpdateClusterFromDB(clusterName, newClusterState, db)
		ut.Assert(t, err == nil, "update cluster from db should succeed: %s", err)

		state, err := getClusterFromDB(clusterName, db)
		ut.Assert(t, err == nil, "get cluster from db should succeed: %s", err)
		ut.Assert(t, state.IsUnvailable == newClusterState.IsUnvailable, "after update cluster, it's IsUnvaliable field should equal the value get from db")

		states, err := getClustersFromDB(db)
		ut.Assert(t, err == nil, "get clusters from db should succeed: %s", err)
		ut.Assert(t, len(states) == 1, "the clusters number that get from db should equal 1")

		err = deleteClusterFromDB(clusterName, db)
		ut.Assert(t, err == nil, "delete cluster from db should succeed: %s", err)

		state, err = getClusterFromDB(clusterName, db)
		ut.Assert(t, err == storage.ErrNotFoundResource, "get cluster from db after delete should get not found err: %s", err)
	})
}
