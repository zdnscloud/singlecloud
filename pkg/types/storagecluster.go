package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetStorageClusterSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type StorageCluster struct {
	resttypes.Resource `json:",inline"`
	Name               string     `json:"name"`
	Namespace          string     `json:"namespace"`
	StorageType        string     `json:"storagetype"`
	Hosts              []HostSpec `json:"hosts"`
	Status             string     `json:"status"`
}

type HostSpec struct {
	NodeName     string   `json:"nodeName"`
	BlockDevices []string `json:"blockDevices"`
}

var StorageClusterType = resttypes.GetResourceType(StorageCluster{})
