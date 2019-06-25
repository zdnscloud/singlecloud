package model

import (
	"github.com/zdnscloud/cement/rstore"
)

const dbFileName = "singlecloud.db"

var gResources []rstore.Resource

func init() {
	gResources = append(gResources, &UserResourceQuota{})
}

var gRStore rstore.ResourceStore

func InitResourceStore() error {
	meta, err := rstore.NewResourceMeta(gResources)
	if err != nil {
		return err
	}

	gRStore, err = rstore.NewRStore(rstore.Sqlite3, map[string]interface{}{"path": dbFileName}, meta)
	return err
}

func Begin() (rstore.Transaction, error) {
	if tx, err := gRStore.Begin(); err != nil {
		return nil, err
	} else {
		return tx, nil
	}
}
