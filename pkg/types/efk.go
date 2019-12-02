package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	DefaultEFKESReplicas   = 3
	DefaultEFKStorageClass = "lvm"
	DefaultEFKStorageSize  = 10
)

type EFK struct {
	resource.ResourceBase `json:",inline"`
	IngressDomain         string `json:"ingressDomain,omitempty"`
	ESReplicas            int    `json:"esReplicas,omitempty"`
	StorageSize           int    `json:"storageSize,omitempty"`
	StorageClass          string `json:"storageClass,omitempty"`
	RedirectUrl           string `json:"redirectUrl,omitempty"`
	Status                string `json:"status,omitempty"`
}

func (e EFK) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (e EFK) CreateDefaultResource() resource.Resource {
	return &EFK{
		StorageClass: DefaultEFKStorageClass,
		StorageSize:  DefaultEFKStorageSize,
		ESReplicas:   DefaultEFKESReplicas,
	}
}
