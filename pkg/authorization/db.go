package authorization

import (
	"encoding/json"

	"github.com/zdnscloud/singlecloud/storage"
)

var (
	AuthorizerTableName = "authorizer"
)

func (a *Authorizer) loadUsers(db storage.DB) error {
	table, err := db.CreateOrGetTable(storage.GenTableName(AuthorizerTableName))
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
	a.users = users
	a.db = table
	return nil
}

func (a *Authorizer) addUser(name string, user *User) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	if err := tx.Add(name, data); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *Authorizer) deleteUser(userName string) error {
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

func (a *Authorizer) updateUser(name string, user *User) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	if err := tx.Update(name, data); err != nil {
		return err
	}
	return tx.Commit()
}
