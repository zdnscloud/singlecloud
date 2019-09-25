package types

import (
	"github.com/zdnscloud/gorest/resource"
	corev1 "k8s.io/api/core/v1"
)

var StorageclusterMap = map[string]string{
	"lvm":    "lvm",
	"cephfs": "cephfs",
}
var StorageAccessModeMap = map[string]corev1.PersistentVolumeAccessMode{
	"lvm":    corev1.ReadWriteOnce,
	"cephfs": corev1.ReadWriteMany,
}

type StorageCluster struct {
	resource.ResourceBase `json:",inline"`
	Name                  string        `json:"-"`
	StorageType           string        `json:"storageType" rest:"required=true,options=lvm|cephfs"`
	Hosts                 []string      `json:"hosts" rest:"required=true"`
	Phase                 string        `json:"phase"`
	Size                  string        `json:"size"`
	UsedSize              string        `json:"usedSize"`
	FreeSize              string        `json:"freeSize"`
	Nodes                 []StorageNode `json:"nodes"`
	PVs                   []PV          `json:"pvs"`
}

type Storage struct {
	Name string `json:"name"`
	PVs  []PV   `json:"pvs"`
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

type StorageNodes []StorageNode

func (s StorageNodes) Len() int           { return len(s) }
func (s StorageNodes) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s StorageNodes) Less(i, j int) bool { return s[i].Name < s[j].Name }
