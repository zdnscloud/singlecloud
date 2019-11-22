package authorization

import (
	"os"
	"testing"
	"time"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/kvzoo/backend/bolt"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func TestUserDB(t *testing.T) {
	dbPath := "user.db"
	ut.WithTempFile(t, dbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		auth, err := New(db)
		ut.Assert(t, err == nil, "load user should succeed")
		ut.Assert(t, auth.HasUser(types.Administrator), "")

		newUser := &types.User{
			Name: "ben",
			Projects: []types.Project{
				types.Project{
					Cluster:   "local",
					Namespace: "default",
				},
			},
		}
		newUser.SetID(newUser.Name)
		createTime := time.Now()
		newUser.SetCreationTimestamp(createTime)
		auth.AddUser(newUser)

		auth, err = New(db)
		ut.Assert(t, err == nil, "load user should succeed")
		ut.Assert(t, auth.HasUser(types.Administrator), "")
		ut.Assert(t, auth.HasUser(newUser.Name), "")
		ut.Assert(t, auth.HasUser(newUser.Name), "")
		ut.Assert(t, auth.Authorize(newUser.Name, "local", "default"), "")
		ben := auth.GetUser("ben")
		ut.Equal(t, ben.GetCreationTimestamp().Second(), createTime.Second())

		newUser.Projects = []types.Project{}
		err = auth.UpdateUser(newUser)
		ut.Assert(t, err == nil, "update user should succeed")

		auth, err = New(db)
		ut.Assert(t, err == nil, "load user should succeed")
		ut.Assert(t, auth.HasUser(newUser.Name), "")
		ut.Assert(t, auth.Authorize(newUser.Name, "local", "default") == false, "")
	})
}
