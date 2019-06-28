package rstore

import (
	"database/sql"
)

type ResourceStore interface {
	Clean()
	Destroy()
	Begin() (Transaction, error)
	BeginTx(sql.IsolationLevel) (Transaction, error)
}

//	a resource array like ([]zone) is different from
//	the resource interface array ([]resource)
//	Get will return the concreate resource array
//	Fill will accept the pointer to the concreate resource array
//	Delete and Update will return how many rows has been affected
type Transaction interface {
	Insert(r Resource) (Resource, error)
	Get(typ ResourceType, cond map[string]interface{}) (interface{}, error)
	Exists(typ ResourceType, cond map[string]interface{}) (bool, error)
	Count(typ ResourceType, cond map[string]interface{}) (int64, error)
	Fill(cond map[string]interface{}, out interface{}) error
	Delete(typ ResourceType, cond map[string]interface{}) (int64, error)
	Update(typ ResourceType, nv map[string]interface{}, cond map[string]interface{}) (int64, error)
	GetOwned(owner ResourceType, ownerID string, owned ResourceType) (interface{}, error)
	FillOwned(owner ResourceType, ownerID string, out interface{}) error

	GetEx(typ ResourceType, sql string, params ...interface{}) (interface{}, error)
	CountEx(typ ResourceType, sql string, params ...interface{}) (int64, error)
	FillEx(out interface{}, sql string, params ...interface{}) error
	DeleteEx(sql string, params ...interface{}) (int64, error)
	Exec(sql string, params ...interface{}) (int64, error)

	GetDefaultResource(typ ResourceType) (Resource, error)

	Commit() error
	RollBack() error
}

func WithTx(store ResourceStore, f func(Transaction) error) error {
	tx, err := store.Begin()
	if err == nil {
		err = f(tx)
		if err == nil {
			tx.Commit()
		} else {
			tx.RollBack()
		}
	}
	return err
}
