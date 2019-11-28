package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type PersistentVolume struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"description=readonly"`
	StorageSize           string   `json:"storageSize" rest:"description=readonly"`
	StorageClassName      string   `json:"storageClassName" rest:"description=readonly"`
	ClaimRef              ClaimRef `json:"claimRef" rest:"description=readonly"`
	Status                string   `json:"status" rest:"description=readonly"`
}

type ClaimRef struct {
	Kind      string `json:"string" rest:"description=readonly"`
	Name      string `json:"name" rest:"description=readonly"`
	Namespace string `json:"namespace" rest:"description=readonly"`
}

func (pv PersistentVolume) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
