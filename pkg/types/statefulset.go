package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	StorageClassNameLVM    = "lvm"
	StorageClassNameCephfs = "cephfs"
	StorageClassNameTemp   = "temporary"
)

func SetStatefulSetSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE", "POST"}
	schema.Parents = []string{NamespaceType}
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name: ActionGetHistory,
	})
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionRollback,
		Input: RollBackVersion{},
	})
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionSetImage,
		Input: SetImage{},
	})
}

type StatefulSet struct {
	resttypes.Resource `json:",inline"`
	Name               string                     `json:"name,omitempty"`
	Replicas           int                        `json:"replicas"`
	Containers         []Container                `json:"containers"`
	AdvancedOptions    AdvancedOptions            `json:"advancedOptions"`
	PersistentVolumes  []PersistentVolumeTemplate `json:"persistentVolumes"`
	Status             WorkloadStatus             `json:"status,omitempty"`
}

type PersistentVolumeTemplate struct {
	Name             string `json:"name"`
	Size             string `json:"size"`
	StorageClassName string `json:"storageClassName"`
}

var StatefulSetType = resttypes.GetResourceType(StatefulSet{})
