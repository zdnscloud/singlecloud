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
	Name               string      `json:"name"`
	StorageType        string      `json:"storagetype"`
	Hosts              []HostSpec  `json:"hosts"`
	Status             ClusterInfo `json:"status"`
}

type HostSpec struct {
	NodeName     string   `json:"nodeName"`
	BlockDevices []string `json:"blockDevices"`
}

type ClusterInfo struct {
	Health string  `json:"health"`
	Info   Storage `json:"info"`
}

var StorageClusterType = resttypes.GetResourceType(StorageCluster{})

type Storage struct {
	Name     string        `json:"-"`
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
