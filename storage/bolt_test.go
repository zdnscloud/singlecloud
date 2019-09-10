package storage

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	ut "github.com/zdnscloud/cement/unittest"
)

const (
	dbName    = "teststorage.db"
	tableName = "/test_resource"
)

func TestTable(t *testing.T) {
	db, err := bolt.Open(dbName, 0666, nil)
	ut.Equal(t, err, nil)

	m := &Storage{db: db}
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
	m := &Storage{db: db}
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
	m := &Storage{db: db}
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
	ut.Equal(t, err, ErrNotFoundResource)
	ut.Equal(t, len(value), 0)

	err = tx.Commit()
	ut.Equal(t, err, nil)
}

func addResource(db DB, key, value string) error {
	table, err := db.CreateOrGetTable(tableName)
	if err != nil {
		return err
	}

	tx, err := table.Begin()
	if err != nil {
		return err
	}

	if err := tx.Add(key, []byte(value)); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func TestList(t *testing.T) {
	db, err := bolt.Open(dbName, 0666, nil)
	ut.Equal(t, err, nil)

	m := &Storage{db: db}
	defer func() {
		m.Close()
		os.Remove(dbName)
	}()

	for i := 0; i < 3; i++ {
		go addResource(m, "k"+strconv.Itoa(i), "v"+strconv.Itoa(i))
	}

	time.Sleep(3 * time.Second)

	table, err := m.CreateOrGetTable(tableName)
	ut.Equal(t, err, nil)
	tx, err := table.Begin()
	ut.Equal(t, err, nil)
	defer tx.Commit()
	value, err := tx.List()
	ut.Equal(t, err, nil)
	ut.Equal(t, len(value), 3)

	for k, v := range value {
		switch k {
		case "k0":
			ut.Equal(t, string(v), "v0")
		case "k1":
			ut.Equal(t, string(v), "v1")
		case "k2":
			ut.Equal(t, string(v), "v2")
		default:
			ut.Equal(t, "no found key "+k, "")
		}
	}
}

func TestNestedTable(t *testing.T) {
	db, err := bolt.Open(dbName, 0666, nil)
	ut.Equal(t, err, nil)

	m := &Storage{db: db}
	defer func() {
		m.Close()
		os.Remove(dbName)
	}()

	nestedTableName := "/app/cd/ns1"
	table, err := m.CreateOrGetTable(nestedTableName)
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

	nestedTableName = "/app/cd/ns2"
	table, err = m.CreateOrGetTable(nestedTableName)
	ut.Equal(t, err, nil)
	tx, err = table.Begin()
	ut.Equal(t, err, nil)
	defer tx.Rollback()
	err = tx.Add("foo", []byte("bar"))
	ut.Equal(t, err, nil)
	value, err = tx.Get("foo")
	ut.Equal(t, err, nil)
	ut.Equal(t, string(value), "bar")
	err = tx.Commit()
	ut.Equal(t, err, nil)

	err = m.DeleteTable(nestedTableName)
	ut.Equal(t, err, nil)
	table, err = m.CreateOrGetTable(nestedTableName)
	ut.Equal(t, err, nil)
	tx, err = table.Begin()
	ut.Equal(t, err, nil)
	defer tx.Rollback()
	value, err = tx.Get("foo")
	ut.Equal(t, err, ErrNotFoundResource)
	err = tx.Commit()
	ut.Equal(t, err, nil)

	nestedTableName = "/app/cd"
	err = m.DeleteTable(nestedTableName)
	ut.Equal(t, err, nil)
	nestedTableName = "/app/cd/ns1"
	table, err = m.CreateOrGetTable(nestedTableName)
	ut.Equal(t, err, nil)
	tx, err = table.Begin()
	ut.Equal(t, err, nil)
	defer tx.Rollback()
	value, err = tx.Get("foo")
	ut.Equal(t, err, ErrNotFoundResource)
	err = tx.Commit()
	ut.Equal(t, err, nil)
}
