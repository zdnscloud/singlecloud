package types

import (
	"github.com/zdnscloud/gorest/resource"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
)

type StorageCluster struct {
	resource.ResourceBase `json:",inline"`
	Name                  string               `json:"name" rest:"required=true,minLen=1,maxLen=128"`
	StorageType           string               `json:"storageType" rest:"required=true,options=lvm|ceph"`
	Hosts                 []string             `json:"hosts" rest:"required=true"`
	Config                []storagev1.HostInfo `json:"config"`
	FreeDevs              []*BlockDevice       `json:"freeDevs"`
	Phase                 string               `json:"phase"`
	Size                  string               `json:"size"`
	UsedSize              string               `json:"usedSize"`
	FreeSize              string               `json:"freeSize"`
	Nodes                 []StorageNode        `json:"nodes"`
	PVs                   []PV                 `json:"pvs"`
}

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

func (s StorageCluster) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
