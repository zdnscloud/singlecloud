package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
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

func SetStorageClusterSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	schema.Parents = []string{ClusterType}
}

type StorageCluster struct {
	resttypes.Resource `json:",inline"`
	Name               string        `json:"name"`
	StorageType        string        `json:"storageType" rest:"required=true,options=lvm|cephfs"`
	Hosts              []string      `json:"hosts" rest:"required=true"`
	Phase              string        `json:"phase"`
	Size               string        `json:"size"`
	UsedSize           string        `json:"usedSize"`
	FreeSize           string        `json:"freeSize"`
	Nodes              []StorageNode `json:"nodes"`
	PVs                []PV          `json:"pvs"`
}

var StorageClusterType = resttypes.GetResourceType(StorageCluster{})

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

type StorageNodes []StorageNode

func (s StorageNodes) Len() int           { return len(s) }
func (s StorageNodes) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s StorageNodes) Less(i, j int) bool { return s[i].Name < s[j].Name }
