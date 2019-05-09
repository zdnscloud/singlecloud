package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
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
	ServiceName         string              `json:"serviceName,omitempty"`
	Replicas            int                 `json:"replicas"`
	Containers          []Container         `json:"containers"`
	AdvancedOptions     AdvancedOptions     `json:"advancedOptions"`
	VolumeClaimTemplate VolumeClaimTemplate `json:"volumeClaimTemplate"`
}

type VolumeClaimTemplate struct {
	Name             string `json:"name"`
	MountPath        string `json:"mountPath"`
	StorageSize      string `json:"storageSize"`
	StorageClassName string `json:"storageClassName"`
}

const (
	StorageClassNameLVM  = "lvm"
	StorageClassNameNFS  = "nfs"
	StorageClassNameTemp = "temporary"
)

var StatefulSetType = resttypes.GetResourceType(StatefulSet{})
