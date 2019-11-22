package kvzoo

import (
	"errors"
)

var ErrNotFound = errors.New("key doesn't exist")

type DB interface {
	//footprint of the data
	//normally it will iterate all the key and values
	//calculate the hash
	Checksum() (string, error)
	//Close and Destroy are mutually exclusive
	//release the conn
	Close() error
	//clean all the data and release the conn
	Destroy() error

	//like path, create child table, will create all parent table too
	CreateOrGetTable(TableName) (Table, error)
	//delete parent table will delete all child table
	DeleteTable(TableName) error
}

type Table interface {
	Begin() (Transaction, error)
}

type Transaction interface {
	Commit() error
	Rollback() error

	Add(string, []byte) error
	//delete non-exist key returns nil
	Delete(string) error
	Update(string, []byte) error
	//get non-exist key return ErrNotFound
	Get(string) ([]byte, error)
	List() (map[string][]byte, error)
}
