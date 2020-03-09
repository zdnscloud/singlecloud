package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	LvmType    StorageType = "lvm"
	CephfsType StorageType = "cephfs"
	IscsiType  StorageType = "iscsi"
	NfsType    StorageType = "nfs"
)

type Storage struct {
	resource.ResourceBase `json:",inline"`
	Name                  string      `json:"name" rest:"required=true,isDomain=true"`
	Type                  StorageType `json:"type" rest:"required=true"`
	Parameter             `json:",inline"`
	Default               bool          `json:"default" rest:"description=readonly"`
	Phase                 string        `json:"phase" rest:"description=readonly"`
	Size                  string        `json:"size" rest:"description=readonly"`
	UsedSize              string        `json:"usedSize" rest:"description=readonly"`
	FreeSize              string        `json:"freeSize" rest:"description=readonly"`
	Nodes                 []StorageNode `json:"nodes" rest:"description=readonly"`
	PVs                   []PV          `json:"pvs" rest:"description=readonly"`
}

type StorageType string

type Parameter struct {
	Lvm    StorageClusterParameter `json:"lvm,omitempty"`
	CephFs StorageClusterParameter `json:"cephfs,omitempty"`
	Iscsi  IscsiParameter          `json:"iscsi,omitempty"`
	Nfs    NfsParameter            `json:"nfs,omitempty"`
}

type StorageClusterParameter struct {
	Hosts []string `json:"hosts" rest:"required=true"`
}

type IscsiParameter struct {
	Target     string   `json:"target" rest:"required=true"`
	Port       string   `json:"port" rest:"required=true"`
	Iqn        string   `json:"iqn" rest:"required=true"`
	Chap       bool     `json:"chap"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	Initiators []string `json:"initiators" rest:"required=true"`
}

type NfsParameter struct {
	Server string `json:"server" rest:"required=true"`
	Path   string `json:"path" rest:"required=true"`
}

type PV struct {
	Name             string       `json:"name"`
	Size             string       `json:"size"`
	UsedSize         string       `json:"usedSize"`
	FreeSize         string       `json:"freeSize"`
	Pods             []StoragePod `json:"pods"`
	StorageClassName string       `json:"-"`
	Node             string       `json:"node"`
	PVC              string       `json:"pvc"`
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

type PVInfo struct {
	Name string `json:"name"`
	PVs  []PV   `json:"pvs"`
}

func (s Storage) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (s Storage) SupportAsyncDelete() bool {
	return true
}

type StorageNodes []StorageNode

func (s StorageNodes) Len() int           { return len(s) }
func (s StorageNodes) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s StorageNodes) Less(i, j int) bool { return s[i].Name < s[j].Name }

type Storages []*Storage

func (s Storages) Len() int           { return len(s) }
func (s Storages) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Storages) Less(i, j int) bool { return s[i].Name < s[j].Name }
