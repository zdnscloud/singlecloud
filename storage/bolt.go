package storage

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/boltdb/bolt"
)

const (
	dbFileName  = "singlecloud.db"
	openTimeout = 5 * time.Second
)

var (
	ErrNotFoundResource  = fmt.Errorf("no found resource in db")
	ErrDuplicateResource = fmt.Errorf("duplicate resource in db")
)

type Storage struct {
	db *bolt.DB
}

func New(fileDir string) (DB, error) {
	if fileDir != "" {
		if err := os.MkdirAll(fileDir, os.ModePerm); err != nil {
			return nil, err
		}
	}

	return CreateWithPath(path.Join(fileDir, dbFileName))
}

func CreateWithPath(filePath string) (DB, error) {
	db, err := bolt.Open(filePath, 0664, &bolt.Options{
		Timeout: openTimeout,
	})
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
		return ErrDuplicateResource
	}
	return m.bucket.Put([]byte(key), value)
}

func (m *TableTX) Delete(key string) error {
	return m.bucket.Delete([]byte(key))
}

func (m *TableTX) Update(key string, value []byte) error {
	if v := m.bucket.Get([]byte(key)); v == nil {
		return ErrNotFoundResource
	}

	return m.bucket.Put([]byte(key), value)
}

func (m *TableTX) Get(key string) ([]byte, error) {
	if v := m.bucket.Get([]byte(key)); v != nil {
		tmp := make([]byte, len(v))
		copy(tmp, v)
		return tmp, nil
	} else {
		return nil, ErrNotFoundResource
	}
}

func (m *TableTX) List() (map[string][]byte, error) {
	resourceMap := make(map[string][]byte)
	if err := m.bucket.ForEach(func(k, v []byte) error {
		tmp := make([]byte, len(v))
		copy(tmp, v)
		resourceMap[string(k)] = tmp
		return nil
	}); err != nil {
		return nil, err
	}

	return resourceMap, nil
}
