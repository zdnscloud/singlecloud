package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type PersistentVolumeClaim struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"description=readonly"`
	StorageClassName      string   `json:"storageClassName" rest:"description=readonly"`
	ActualStorageSize     string   `json:"actualStorageSize" rest:"description=readonly"`
	Used                  bool     `json:"used" rest:"description=readonly"`
	Pods                  []string `json:"pods" rest:"description=readonly"`
	Node                  string   `json:"node" rest:"description=readonly"`
	Driver                string   `json:"-"`
	RequestStorageSize    string   `json:"-"`
	VolumeName            string   `json:"-"`
	Status                string   `json:"-"`
}

func (pvc PersistentVolumeClaim) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (pvc PersistentVolumeClaim) SupportAsyncDelete() bool {
	return true
}
