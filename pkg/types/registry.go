package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Registry struct {
	resource.ResourceBase `json:",inline"`
	IngressDomain         string `json:"ingressDomain"`
	StorageClass          string `json:"storageClass"`
	StorageSize           int    `json:"storageSize"`
	AdminPassword         string `json:"adminPassword"`
	RedirectUrl           string `json:"redirectUrl"`
	Status                string `json:"status"`
}

func (r Registry) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
