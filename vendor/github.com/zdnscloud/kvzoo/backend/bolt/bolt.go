package bolt

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	stdpath "path"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/zdnscloud/kvzoo"
)

const (
	openTimeout   = 5 * time.Second
	checkKeyCount = 10
)

var (
	ErrInvalidDBPath     = fmt.Errorf("db file doesn't exist")
	ErrDuplicateResource = fmt.Errorf("duplicate resource")
)

type BoltDB struct {
	path string
	db   *bolt.DB
}

func New(path string) (kvzoo.DB, error) {
	if path == "" {
		return nil, ErrInvalidDBPath
	}

	dir := stdpath.Dir(path)
	if dir != "" {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, err
		}
	}

	db, err := bolt.Open(path, 0664, &bolt.Options{
		Timeout: openTimeout,
	})
	if err != nil {
		return nil, err
	}

	return &BoltDB{
		db:   db,
		path: path,
	}, nil
}

func (db *BoltDB) Checksum() (string, error) {
	tx, err := db.db.Begin(false)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	h := md5.New()
	c := tx.Cursor()
	k, v := c.First()
	if k != nil {
		h.Write(k)
		if v != nil {
			h.Write(v)
		} else {
			db.bucketCheckSum(h, tx.Bucket(k))
		}
		for {
			k, v := c.Next()
			if k == nil {
				break
			}
			h.Write(k)
			if v != nil {
				h.Write(v)
			} else {
				db.bucketCheckSum(h, tx.Bucket(k))
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil)[:16]), nil
}

func (db *BoltDB) bucketCheckSum(h hash.Hash, b *bolt.Bucket) {
	c := b.Cursor()
	k, v := c.First()
	if k != nil {
		h.Write(k)
		if v != nil {
			h.Write(v)
		} else {
			db.bucketCheckSum(h, b.Bucket(k))
		}
		for {
			k, v := c.Next()
			if k == nil {
				break
			}
			h.Write(k)
			if v != nil {
				h.Write(v)
			} else {
				db.bucketCheckSum(h, b.Bucket(k))
			}
		}
	}
}

func (db *BoltDB) Close() error {
	return db.db.Close()
}

func (db *BoltDB) Destroy() error {
	if err := db.Close(); err != nil {
		return err
	}
	return os.Remove(db.path)
}

func (db *BoltDB) CreateOrGetTable(tableName kvzoo.TableName) (kvzoo.Table, error) {
	tx, err := db.db.Begin(true)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	if _, err := createOrGetBucket(tx, string(tableName)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &DBTable{
		name: string(tableName),
		db:   db.db,
	}, nil
}

func (db *BoltDB) DeleteTable(tableName kvzoo.TableName) error {
	tx, err := db.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tables := tableName.Segments()
	if len(tables) == 1 {
		if err := tx.DeleteBucket([]byte(tables[0])); err != nil {
			return err
		}
	} else {
		bucket := tx.Bucket([]byte(tables[0]))
		if bucket == nil {
			return fmt.Errorf("no found table %s", tables[0])
		}

		for i := 1; i < len(tables)-1; i++ {
			if bucket = bucket.Bucket([]byte(tables[i])); bucket == nil {
				return fmt.Errorf("no found table %s", tables[i])
			}
		}
		if err := bucket.DeleteBucket([]byte(tables[len(tables)-1])); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func createOrGetBucket(tx *bolt.Tx, tableName string) (*bolt.Bucket, error) {
	var bucket *bolt.Bucket
	var err error
	for i, table := range strings.Split(strings.TrimPrefix(tableName, "/"), "/") {
		if table == "" {
			return nil, fmt.Errorf("table name %s is invalid, contains empty table name", tableName)
		}

		if i == 0 {
			if bucket, err = tx.CreateBucketIfNotExists([]byte(table)); err != nil {
				return nil, err
			}
		} else {
			if bucket, err = bucket.CreateBucketIfNotExists([]byte(table)); err != nil {
				return nil, err
			}
		}
	}

	return bucket, nil
}

type DBTable struct {
	name string
	db   *bolt.DB
}

func (db *DBTable) Begin() (kvzoo.Transaction, error) {
	tx, err := db.db.Begin(true)
	if err != nil {
		return nil, err
	}

	bucket, err := createOrGetBucket(tx, db.name)
	if err != nil {
		tx.Commit()
		return nil, err
	}

	if bucket == nil {
		tx.Commit()
		return nil, fmt.Errorf("table %s is non-exists", db.name)
	}

	return &TableTX{
		bucket: bucket,
	}, nil
}

type TableTX struct {
	bucket *bolt.Bucket
}

func (tx *TableTX) Rollback() error {
	return tx.bucket.Tx().Rollback()
}

func (tx *TableTX) Commit() error {
	return tx.bucket.Tx().Commit()
}

func (tx *TableTX) Add(key string, value []byte) error {
	if v := tx.bucket.Get([]byte(key)); v != nil {
		return ErrDuplicateResource
	}
	return tx.bucket.Put([]byte(key), value)
}

func (tx *TableTX) Delete(key string) error {
	return tx.bucket.Delete([]byte(key))
}

func (tx *TableTX) Update(key string, value []byte) error {
	if v := tx.bucket.Get([]byte(key)); v == nil {
		return kvzoo.ErrNotFound
	}

	return tx.bucket.Put([]byte(key), value)
}

func (tx *TableTX) Get(key string) ([]byte, error) {
	if v := tx.bucket.Get([]byte(key)); v != nil {
		tmp := make([]byte, len(v))
		copy(tmp, v)
		return tmp, nil
	} else {
		return nil, kvzoo.ErrNotFound
	}
}

func (tx *TableTX) List() (map[string][]byte, error) {
	resourceMap := make(map[string][]byte)
	if err := tx.bucket.ForEach(func(k, v []byte) error {
		tmp := make([]byte, len(v))
		copy(tmp, v)
		resourceMap[string(k)] = tmp
		return nil
	}); err != nil {
		return nil, err
	}

	return resourceMap, nil
}
