package storage

import (
	"fmt"
	"path"

	"github.com/boltdb/bolt"
)

const (
	dbFileName = "singlecloud.db"
)

type Storage struct {
	db *bolt.DB
}

func New(filePath string) (DB, error) {
	db, err := bolt.Open(path.Join(filePath, dbFileName), 0664, nil)
	if err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

func (m *Storage) Close() error {
	return m.db.Close()
}

func (m *Storage) CreateOrGetTable(tableName string) (Table, error) {
	tx, err := m.db.Begin(true)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()
	if _, err := tx.CreateBucketIfNotExists([]byte(tableName)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &DBTable{
		name: tableName,
		db:   m.db,
	}, nil
}

func (m *Storage) DeleteTable(tableName string) error {
	tx, err := m.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.DeleteBucket([]byte(tableName)); err != nil {
		return err
	}

	return tx.Commit()
}

type DBTable struct {
	name string
	db   *bolt.DB
}

func (m *DBTable) Begin() (Transaction, error) {
	tx, err := m.db.Begin(true)
	if err != nil {
		return nil, err
	}

	bucket := tx.Bucket([]byte(m.name))
	if bucket == nil {
		tx.Commit()
		return nil, fmt.Errorf("table %s is non-exists", m.name)
	}

	return &TableTX{
		bucket: bucket,
	}, nil
}

type TableTX struct {
	bucket *bolt.Bucket
}

func (m *TableTX) Rollback() error {
	return m.bucket.Tx().Rollback()
}

func (m *TableTX) Commit() error {
	return m.bucket.Tx().Commit()
}

func (m *TableTX) Add(key string, value []byte) error {
	if v := m.bucket.Get([]byte(key)); v != nil {
		return fmt.Errorf("duplicate resource %s", key)
	}
	return m.bucket.Put([]byte(key), value)
}

func (m *TableTX) Delete(key string) error {
	return m.bucket.Delete([]byte(key))
}

func (m *TableTX) Update(key string, value []byte) error {
	if v := m.bucket.Get([]byte(key)); v == nil {
		return fmt.Errorf("no found resource by key %s", key)
	}

	return m.bucket.Put([]byte(key), value)
}

func (m *TableTX) Get(key string) ([]byte, error) {
	if v := m.bucket.Get([]byte(key)); v != nil {
		return v, nil
	} else {
		return nil, fmt.Errorf("no found resource by key %s", key)
	}
}

func (m *TableTX) List() (map[string][]byte, error) {
	resourceMap := make(map[string][]byte)
	if err := m.bucket.ForEach(func(k, v []byte) error {
		resourceMap[string(k)] = v
		return nil
	}); err != nil {
		return nil, err
	}

	return resourceMap, nil
}
