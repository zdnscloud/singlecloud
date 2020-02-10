package jwt

import (
	"os"
	"testing"

	"github.com/zdnscloud/kvzoo"
	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/kvzoo/backend/bolt"
	"github.com/zdnscloud/singlecloud/pkg/authentication/session"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func newAuthenticator(db kvzoo.DB) (*Authenticator, error) {
	auth := &Authenticator{
		repo:     NewTokenRepo(tokenSecret, tokenValidDuration),
		sessions: session.New(SessionCookieName),
	}

	if err := loadUsersFromDB(db, auth); err != nil {
		return nil, err
	}

	if _, ok := auth.users[types.Administrator]; ok == false {
		admin := &types.User{
			Name:     types.Administrator,
			Password: AdminPasswd,
		}
		admin.SetID(types.Administrator)
		auth.AddUser(admin)
	}

	return auth, nil
}

func loadUsersFromDB(db kvzoo.DB, auth *Authenticator) error {
	tn, _ := kvzoo.TableNameFromSegments(JwtAuthenticatorTableName)
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

	users := make(map[string]string)
	for name, pwd := range usersInDB {
		users[name] = string(pwd)
	}

	auth.users = users
	auth.db = table
	return nil
}

func TestUserDB(t *testing.T) {
	dbPath := "user.db"
	ut.WithTempFile(t, dbPath, func(t *testing.T, f *os.File) {
		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed:%v", err)
		auth, err := newAuthenticator(db)
		ut.Assert(t, err == nil, "load user should succeed:%v", err)
		ut.Assert(t, auth.HasUser(types.Administrator), "")

		newUser := &types.User{
			Name:     "ben",
			Password: "123",
		}
		newUser.SetID(newUser.Name)
		auth.AddUser(newUser)
		ut.Assert(t, auth.HasUser(newUser.Name), "")

		auth, err = newAuthenticator(db)
		ut.Assert(t, err == nil, "load user should succeed")
		ut.Assert(t, auth.HasUser(types.Administrator), "")
		ut.Assert(t, auth.HasUser(newUser.Name), "")
		err = auth.ResetPassword(newUser.Name, newUser.Password, "345", false)
		ut.Assert(t, err == nil, "reset user password should succeed")
		err = auth.ResetPassword(types.Administrator, AdminPasswd, "123", false)
		ut.Assert(t, err == nil, "reset admin password should succeed, %v", err)
		err = auth.DeleteUser(newUser.Name)
		ut.Assert(t, err == nil, "delete user should succeed")

		auth, err = newAuthenticator(db)
		ut.Assert(t, err == nil, "load user should succeed")
		ut.Assert(t, auth.HasUser(types.Administrator), "")
		ut.Assert(t, auth.HasUser(newUser.Name) == false, "")
		ut.Equal(t, auth.users[types.Administrator], "123")
	})
}
