package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Registry struct {
	resource.ResourceBase `json:",inline"`
	IngressDomain         string `json:"ingressDomain" rest:"description=immutable,isDomain=true"`
	StorageClass          string `json:"storageClass" rest:"description=immutable"`
	StorageSize           int    `json:"storageSize" rest:"description=immutable"`
	AdminPassword         string `json:"adminPassword" rest:"description=immutable"`
	RedirectUrl           string `json:"redirectUrl" rest:"description=readonly"`
	Status                string `json:"status" rest:"description=readonly"`
}

func (r Registry) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
