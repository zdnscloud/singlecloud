package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type PersistentVolumeClaim struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name"`
	Namespace             string `json:"namespace"`
	RequestStorageSize    string `json:"requestStorageSize"`
	StorageClassName      string `json:"storageClassName"`
	VolumeName            string `json:"volumeName"`
	ActualStorageSize     string `json:"actualStorageSize"`
	Status                string `json:"status"`
}

func (pvc PersistentVolumeClaim) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
