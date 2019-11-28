package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	defaultEFKESReplicas   = 3
	defaultEFKStorageClass = "lvm"
	defaultEFKStorageSize  = 10
)

type EFK struct {
	resource.ResourceBase `json:",inline"`
	IngressDomain         string `json:"ingressDomain,omitempty"`
	ESReplicas            int    `json:"esReplicas,omitempty"`
	StorageSize           int    `json:"storageSize,omitempty" rest:"description=immutable"`
	StorageClass          string `json:"storageClass,omitempty" rest:"description=immutable"`
	RedirectUrl           string `json:"redirectUrl,omitempty"`
	Status                string `json:"status,omitempty" rest:"description=readonly"`
}

func (e EFK) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (e EFK) CreateDefaultResource() resource.Resource {
	return &EFK{
		StorageClass: defaultEFKStorageClass,
		StorageSize:  defaultEFKStorageSize,
		ESReplicas:   defaultEFKESReplicas,
	}
}
