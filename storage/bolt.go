package storage

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

const (
	dbFileName  = "singlecloud.db"
	openTimeout = 5 * time.Second
	Root        = "/"
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
	if err := checkTableNameValid(tableName); err != nil {
		return nil, err
	}

	tx, err := m.db.Begin(true)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	if _, err := createOrGetBucket(tx, tableName); err != nil {
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
	if err := checkTableNameValid(tableName); err != nil {
		return err
	}

	tx, err := m.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tables := strings.Split(strings.TrimPrefix(tableName, "/"), "/")
	if len(tables) == 1 {
		if err := tx.DeleteBucket([]byte(tables[0])); err != nil {
			return err
		}
	} else {
		rootBucket := tx.Bucket([]byte(tables[0]))
		if rootBucket == nil {
			return fmt.Errorf("no found table %s", tables[0])
		}

		for i := 1; i < len(tables)-1; i++ {
			if bucket := rootBucket.Bucket([]byte(tables[i])); bucket == nil {
				return fmt.Errorf("no found table %s", tables[i])
			} else {
				rootBucket = bucket
			}
		}
		if err := rootBucket.DeleteBucket([]byte(tables[len(tables)-1])); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func checkTableNameValid(tableName string) error {
	if tableName == "" || tableName == "/" {
		return fmt.Errorf("table name should not be empty")
	}

	if strings.HasPrefix(tableName, "/") == false {
		return fmt.Errorf("table name should begin with /")
	}

	return nil
}

func createOrGetBucket(tx *bolt.Tx, tableName string) (*bolt.Bucket, error) {
	var bucket *bolt.Bucket
	for i, table := range strings.Split(strings.TrimPrefix(tableName, "/"), "/") {
		if table == "" {
			return nil, fmt.Errorf("table name %s is invalid, contains empty table name", tableName)
		}

		if i == 0 {
			if rootBucket, err := tx.CreateBucketIfNotExists([]byte(table)); err != nil {
				return nil, err
			} else {
				bucket = rootBucket
			}
		} else {
			if subBucket, err := bucket.CreateBucketIfNotExists([]byte(table)); err != nil {
				return nil, err
			} else {
				bucket = subBucket
			}
		}
	}

	return bucket, nil
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

	bucket, err := createOrGetBucket(tx, m.name)
	if err != nil {
		tx.Commit()
		return nil, err
	}

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

func GenTableName(tables ...string) string {
	return path.Join(append([]string{Root}, tables...)...)
}
