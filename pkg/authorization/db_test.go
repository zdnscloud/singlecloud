package authorization

import (
	"os"
	"testing"
	"time"
	"encoding/json"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/kvzoo/backend/bolt"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/kvzoo"
)

func newAuth(db kvzoo.DB)(*Authorizer, error) {
	auth := &Authorizer{
		users: make(map[string]*User),
	}

	if err := loadUsersFromDB(db, auth); err != nil {
		return nil, err
	}

	if _, ok := auth.users[types.Administrator]; ok == false {
		adminUser.SetID(types.Administrator)
		adminUser.SetCreationTimestamp(zcloudStartTime)
		auth.AddUser(adminUser)
	}

	return auth, nil
}

func loadUsersFromDB(db kvzoo.DB, auth *Authorizer) error {
	tn, _ := kvzoo.TableNameFromSegments(AuthorizerTableName)
	table, err := db.CreateOrGetTable(tn)
	if err != nil {
		return err
	}

	tx, err := table.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	usersInDB, err := tx.List()
	if err != nil {
		return err
	}

	users := make(map[string]*User)
	for name, userInDB := range usersInDB {
		var user User
		if err := json.Unmarshal(userInDB, &user); err != nil {
			return err
		}
		users[name] = &user
	}
	auth.users = users
	auth.db = table
	return nil
}

func TestUserDB(t *testing.T) {
	dbPath := "user.db"
	ut.WithTempFile(t, dbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)
		auth, err := newAuth(db)
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

		auth, err = newAuth(db)
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

		auth, err = newAuth(db)
		ut.Assert(t, err == nil, "load user should succeed")
		ut.Assert(t, auth.HasUser(newUser.Name), "")
		ut.Assert(t, auth.Authorize(newUser.Name, "local", "default") == false, "")
	})
}
