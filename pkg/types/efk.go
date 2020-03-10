package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	DefaultEFKESReplicas = 3
)

type EFK struct {
	resource.ResourceBase `json:",inline"`
	IngressDomain         string `json:"ingressDomain,omitempty" rest:"isDomain=true"`
	ESReplicas            int    `json:"esReplicas,omitempty"`
	StorageSize           int    `json:"storageSize,omitempty" rest:"description=immutable,min=3"`
	StorageClass          string `json:"storageClass,omitempty" rest:"description=immutable"`
	RedirectUrl           string `json:"redirectUrl,omitempty"`
	Status                string `json:"status,omitempty" rest:"description=readonly"`
}

func (e EFK) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
