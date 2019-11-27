package jwt

import (
	"os"
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/kvzoo/backend/bolt"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func TestUserDB(t *testing.T) {
	dbPath := "user.db"
	ut.WithTempFile(t, dbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed:%v", err)
		auth, err := NewAuthenticator(db)
		ut.Assert(t, err == nil, "load user should succeed:%v", err)
		ut.Assert(t, auth.HasUser(types.Administrator), "")

		newUser := &types.User{
			Name:     "ben",
			Password: "123",
		}
		newUser.SetID(newUser.Name)
		auth.AddUser(newUser)
		ut.Assert(t, auth.HasUser(newUser.Name), "")

		auth, err = NewAuthenticator(db)
		ut.Assert(t, err == nil, "load user should succeed")
		ut.Assert(t, auth.HasUser(types.Administrator), "")
		ut.Assert(t, auth.HasUser(newUser.Name), "")
		err = auth.ResetPassword(newUser.Name, newUser.Password, "345", false)
		ut.Assert(t, err == nil, "reset user password should succeed")
		err = auth.ResetPassword(types.Administrator, AdminPasswd, "123", false)
		ut.Assert(t, err == nil, "reset admin password should succeed, %v", err)
		err = auth.DeleteUser(newUser.Name)
		ut.Assert(t, err == nil, "delete user should succeed")

		auth, err = NewAuthenticator(db)
		ut.Assert(t, err == nil, "load user should succeed")
		ut.Assert(t, auth.HasUser(types.Administrator), "")
		ut.Assert(t, auth.HasUser(newUser.Name) == false, "")
		ut.Equal(t, auth.users[types.Administrator], "123")
	})
}
