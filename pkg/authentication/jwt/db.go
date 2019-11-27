package jwt

import (
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var (
	JwtAuthenticatorTableName = "jwt_authenticator"
)

func (a *Authenticator) loadUsers(db kvzoo.DB) error {
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
	a.users = users

	a.db = table
	return nil
}

func (a *Authenticator) addUser(user *types.User) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Add(user.GetID(), []byte(user.Password)); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *Authenticator) deleteUser(userName string) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Delete(userName); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *Authenticator) updateUser(userName string, password string) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Update(userName, []byte(password)); err != nil {
		return err
	}
	return tx.Commit()
}
