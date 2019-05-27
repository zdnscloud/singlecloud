package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	StorageClassNameLVM  = "lvm"
	StorageClassNameNFS  = "nfs"
	StorageClassNameCeph = "ceph"
	StorageClassNameTemp = "temporary"
)

func SetStatefulSetSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type StatefulSet struct {
	resttypes.Resource  `json:",inline"`
	Name                string              `json:"name,omitempty"`
	Replicas            int                 `json:"replicas"`
	Containers          []Container         `json:"containers"`
	AdvancedOptions     AdvancedOptions     `json:"advancedOptions"`
	VolumeClaimTemplate VolumeClaimTemplate `json:"volumeClaimTemplate"`
}

type VolumeClaimTemplate struct {
	Name             string `json:"name"`
	Size             string `json:"size"`
	StorageClassName string `json:"storageClassName"`
}

var StatefulSetType = resttypes.GetResourceType(StatefulSet{})
