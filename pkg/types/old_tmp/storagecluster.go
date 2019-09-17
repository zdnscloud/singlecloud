package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
)

func SetStorageClusterSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	schema.Parents = []string{ClusterType}
}

type StorageCluster struct {
	resttypes.Resource `json:",inline"`
	Name               string               `json:"name" rest:"required=true"`
	StorageType        string               `json:"storageType" rest:"required=true,options=lvm|ceph"`
	Hosts              []string             `json:"hosts" rest:"required=true"`
	Config             []storagev1.HostInfo `json:"config"`
	FreeDevs           []BlockDevice        `json:"freeDevs"`
	Phase              string               `json:"phase"`
	Size               string               `json:"size"`
	UsedSize           string               `json:"usedSize"`
	FreeSize           string               `json:"freeSize"`
	Nodes              []StorageNode        `json:"nodes"`
	PVs                []PV                 `json:"pvs"`
}

var StorageClusterType = resttypes.GetResourceType(StorageCluster{})

type Storage struct {
	Name     string        `json:"name"`
	Size     string        `json:"size"`
	UsedSize string        `json:"usedSize"`
	FreeSize string        `json:"freeSize"`
	Nodes    []StorageNode `json:"nodes"`
	PVs      []PV          `json:"pvs"`
}

type PV struct {
	Name             string       `json:"name"`
	Size             string       `json:"size"`
	UsedSize         string       `json:"usedSize"`
	FreeSize         string       `json:"freeSize"`
	Pods             []StoragePod `json:"pods"`
	StorageClassName string       `json:"-"`
	Node             string       `json:"node"`
}

type StorageNode struct {
	Name     string `json:"name"`
	Size     string `json:"size"`
	UsedSize string `json:"usedSize"`
	FreeSize string `json:"freeSize"`
	Stat     bool   `json:"stat"`
}

type StoragePod struct {
	Name string `json:"name"`
}
