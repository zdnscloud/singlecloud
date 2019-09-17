package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetStorageClassSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parents = []string{ClusterType}
}

type StorageClass struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
}

var StorageClassType = resttypes.GetResourceType(StorageClass{})
