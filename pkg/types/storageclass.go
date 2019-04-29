package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetStorageClassSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parent = ClusterType
}

type StorageClass struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
}

var StorageClassType = resttypes.GetResourceType(StorageClass{})
