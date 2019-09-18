package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type PersistentVolume struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name"`
	StorageSize           string   `json:"storageSize"`
	StorageClassName      string   `json:"storageClassName"`
	ClaimRef              ClaimRef `json:"claimRef"`
	Status                string   `json:"status"`
}

type ClaimRef struct {
	Kind      string `json:"string"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (pv PersistentVolume) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
