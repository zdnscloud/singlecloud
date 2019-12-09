package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type PersistentVolumeClaim struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name" rest:"description=readonly"`
	Namespace             string `json:"namespace" rest:"description=readonly"`
	RequestStorageSize    string `json:"requestStorageSize" rest:"description=readonly"`
	StorageClassName      string `json:"storageClassName" rest:"description=readonly"`
	VolumeName            string `json:"volumeName" rest:"description=readonly"`
	ActualStorageSize     string `json:"actualStorageSize" rest:"description=readonly"`
	Status                string `json:"status" rest:"description=readonly"`
}

func (pvc PersistentVolumeClaim) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
