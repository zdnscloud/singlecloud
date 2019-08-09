package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetStorageClusterSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	schema.Parents = []string{ClusterType}
}

type StorageCluster struct {
	resttypes.Resource `json:",inline"`
	Name               string        `json:"name" rest:"required=true"`
	StorageType        string        `json:"storagetype" rest:"required=true,options=lvm|ceph"`
	Hosts              []string      `json:"hosts" rest:"required=true"`
	Phase              string        `json:"phase"`
	Size               string        `json:"size"`
	UsedSize           string        `json:"usedsize"`
	FreeSize           string        `json:"freesize"`
	Nodes              []StorageNode `json:"nodes"`
	PVs                []PV          `json:"pvs"`
}

var StorageClusterType = resttypes.GetResourceType(StorageCluster{})

type Storage struct {
	Name     string        `json:"name"`
	Size     string        `json:"size"`
	UsedSize string        `json:"usedsize"`
	FreeSize string        `json:"freesize"`
	Nodes    []StorageNode `json:"nodes"`
	PVs      []PV          `json:"pvs"`
}

type PV struct {
	Name             string       `json:"name"`
	Size             string       `json:"size"`
	UsedSize         string       `json:"usedsize"`
	FreeSize         string       `json:"freesize"`
	Pods             []StoragePod `json:"pods"`
	StorageClassName string       `json:"-"`
	Node             string       `json:"node"`
}

type StorageNode struct {
	Name     string `json:"name"`
	Size     string `json:"size"`
	UsedSize string `json:"usedsize"`
	FreeSize string `json:"freesize"`
}

type StoragePod struct {
	Name string `json:"name"`
}