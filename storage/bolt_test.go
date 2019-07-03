package storage

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	ut "github.com/zdnscloud/cement/unittest"
)

type TestResource struct {
	Name string `json:"name"`
}

const (
	dbName    = "teststorage.db"
	tableName = "test_resource"
)

func TestTable(t *testing.T) {
	db, err := bolt.Open(dbName, 0666, nil)
	ut.Equal(t, err, nil)

	m := &StorageManager{db: db}
	defer func() {
		m.Close()
		os.Remove(dbName)
	}()

	table, err := m.CreateOrGetTable(tableName)
	ut.Equal(t, err, nil)

	tx, err := table.Begin()
	ut.Equal(t, err, nil)

	err = tx.Commit()
	ut.Equal(t, err, nil)

	err = m.DeleteTable(tableName)
	ut.Equal(t, err, nil)
}

func TestAddAndGet(t *testing.T) {
	db, err := bolt.Open(dbName, 0666, nil)
	ut.Equal(t, err, nil)
	m := &StorageManager{db: db}
	defer func() {
		m.Close()
		os.Remove(dbName)
	}()

	table, err := m.CreateOrGetTable(tableName)
	ut.Equal(t, err, nil)

	tx, err := table.Begin()
	ut.Equal(t, err, nil)
	defer tx.Rollback()

	err = tx.Add("foo", []byte("bar"))
	ut.Equal(t, err, nil)

	value, err := tx.Get("foo")
	ut.Equal(t, err, nil)
	ut.Equal(t, string(value), "bar")

	err = tx.Commit()
	ut.Equal(t, err, nil)
}

func TestUpdateAndDelete(t *testing.T) {
	db, err := bolt.Open(dbName, 0666, nil)
	ut.Equal(t, err, nil)
	m := &StorageManager{db: db}
	defer func() {
		m.Close()
		os.Remove(dbName)
	}()

	table, err := m.CreateOrGetTable(tableName)
	ut.Equal(t, err, nil)

	tx, err := table.Begin()
	ut.Equal(t, err, nil)
	defer tx.Rollback()

	err = tx.Add("foo", []byte("bar"))
	ut.Equal(t, err, nil)

	value, err := tx.Get("foo")
	ut.Equal(t, err, nil)
	ut.Equal(t, string(value), "bar")

	err = tx.Update("foo", []byte("par"))
	ut.Equal(t, err, nil)

	value, err = tx.Get("foo")
	ut.Equal(t, err, nil)
	ut.Equal(t, string(value), "par")

	err = tx.Delete("foo")
	ut.Equal(t, err, nil)

	value, err = tx.Get("foo")
	ut.Equal(t, err.Error(), "no found resource by key foo")
	ut.Equal(t, len(value), 0)

	err = tx.Commit()
	ut.Equal(t, err, nil)
}

func TestList(t *testing.T) {
	db, err := bolt.Open(dbName, 0666, nil)
	ut.Equal(t, err, nil)

	m := &StorageManager{db: db}
	defer func() {
		m.Close()
		os.Remove(dbName)
	}()

	table, err := m.CreateOrGetTable(tableName)
	ut.Equal(t, err, nil)

	tx, err := table.Begin()
	ut.Equal(t, err, nil)
	defer tx.Rollback()

	err = tx.Add("aoo", []byte("ar"))
	ut.Equal(t, err, nil)
	err = tx.Add("boo", []byte("br"))
	ut.Equal(t, err, nil)
	err = tx.Add("coo", []byte("cr"))
	ut.Equal(t, err, nil)

	value, err := tx.List()
	ut.Equal(t, err, nil)

	for k, v := range value {
		switch k {
		case "aoo":
			ut.Equal(t, string(v), "ar")
		case "boo":
			ut.Equal(t, string(v), "br")
		case "coo":
			ut.Equal(t, string(v), "cr")
		}
	}

	err = tx.Commit()
	ut.Equal(t, err, nil)
}
