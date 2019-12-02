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
	IngressDomain         string `json:"ingressDomain" rest:"description=immutable"`
	StorageClass          string `json:"storageClass" rest:"options=lvm|cephfs,description=immutable"`
	StorageSize           int    `json:"storageSize" rest:"description=immutable"`
	AdminPassword         string `json:"adminPassword" rest:"description=immutable"`
	RedirectUrl           string `json:"redirectUrl" rest:"description=readonly"`
	Status                string `json:"status" rest:"description=readonly"`
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
