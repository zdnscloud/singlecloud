package authorization

import (
	"encoding/json"

	"github.com/zdnscloud/singlecloud/pkg/types"
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

	users := make(map[string]Projects)
	for name, projects_ := range usersInDB {
		var projects Projects
		if err := json.Unmarshal(projects_, &projects); err != nil {
			return err
		}
		users[name] = projects
	}
	a.users = users

	a.db = table
	return nil
}

func (a *Authorizer) addUser(user *types.User) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	data, err := json.Marshal(user.Projects)
	if err != nil {
		return err
	}

	if err := tx.Add(user.GetID(), data); err != nil {
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

func (a *Authorizer) updateUser(user *types.User) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	data, err := json.Marshal(user.Projects)
	if err != nil {
		return err
	}

	if err := tx.Update(user.GetID(), data); err != nil {
		return err
	}
	return tx.Commit()
}
