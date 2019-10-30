package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	DefaultRegistryStorageClass  = "lvm"
	DefaultRegistryStorageSize   = 50
	DefaultRegistryAdminPassWord = "zcloud"
)

type Registry struct {
	resource.ResourceBase `json:",inline"`
	IngressDomain         string `json:"ingressDomain"`
	StorageClass          string `json:"storageClass" rest:"options=lvm|cephfs"`
	StorageSize           int    `json:"storageSize"`
	AdminPassword         string `json:"adminPassword"`
	RedirectUrl           string `json:"redirectUrl"`
	Status                string `json:"status"`
}

func (r Registry) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (r Registry) CreateDefaultResource() resource.Resource {
	return &Registry{
		StorageClass:  DefaultRegistryStorageClass,
		StorageSize:   DefaultRegistryStorageSize,
		AdminPassword: DefaultRegistryAdminPassWord,
	}
}
