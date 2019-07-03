package storage

import (
	"fmt"
	"path"

	"github.com/boltdb/bolt"
)

const (
	dbFileName = "singlecloud.db"
)

type StorageManager struct {
	db *bolt.DB
}

func New(filePath string) (*StorageManager, error) {
	db, err := bolt.Open(path.Join(filePath, dbFileName), 0664, nil)
	if err != nil {
		return nil, err
	}

	return &StorageManager{db: db}, nil
}

func (m *StorageManager) Close() error {
	return m.db.Close()
}

func (m *StorageManager) CreateOrGetTable(tableName string) (Table, error) {
	tx, err := m.db.Begin(true)
	if err != nil {
		return nil, err
	}

	if _, err := tx.CreateBucketIfNotExists([]byte(tableName)); err != nil {
		tx.Rollback()
		return nil, err
	}

	return &TableManager{
		Bucket: tx.Bucket([]byte(tableName)),
	}, nil
}

func (m *StorageManager) DeleteTable(tableName string) error {
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

type TableManager struct {
	Bucket *bolt.Bucket
}

func (m *TableManager) Begin() (Transaction, error) {
	return &TransactionManager{
		Bucket: m.Bucket,
	}, nil
}

type TransactionManager struct {
	Bucket *bolt.Bucket
}

func (m *TransactionManager) Rollback() error {
	return m.Bucket.Tx().Rollback()
}

func (m *TransactionManager) Commit() error {
	return m.Bucket.Tx().Commit()
}

func (m *TransactionManager) Add(key string, value []byte) error {
	if v := m.Bucket.Get([]byte(key)); v != nil {
		return fmt.Errorf("duplicate resource %s", key)
	}
	return m.Bucket.Put([]byte(key), value)
}

func (m *TransactionManager) Delete(key string) error {
	return m.Bucket.Delete([]byte(key))
}

func (m *TransactionManager) Update(key string, value []byte) error {
	if v := m.Bucket.Get([]byte(key)); v == nil {
		return fmt.Errorf("no found resource by key %s", key)
	}

	return m.Bucket.Put([]byte(key), value)
}

func (m *TransactionManager) Get(key string) ([]byte, error) {
	if v := m.Bucket.Get([]byte(key)); v != nil {
		return v, nil
	} else {
		return nil, fmt.Errorf("no found resource by key %s", key)
	}
}

func (m *TransactionManager) List() (map[string][]byte, error) {
	resourceMap := make(map[string][]byte)
	if err := m.Bucket.ForEach(func(k, v []byte) error {
		resourceMap[string(k)] = v
		return nil
	}); err != nil {
		return nil, err
	}

	return resourceMap, nil
}
