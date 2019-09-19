package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	StorageClassNameLVM  = "lvm"
	StorageClassNameCeph = "cephfs"
	StorageClassNameTemp = "temporary"
)

type StatefulSet struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                     `json:"name,omitempty"`
	Replicas              int                        `json:"replicas"`
	Containers            []Container                `json:"containers"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes"`
}

type PersistentVolumeTemplate struct {
	Name             string `json:"name"`
	Size             string `json:"size"`
	StorageClassName string `json:"storageClassName"`
}

func (s StatefulSet) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (s StatefulSet) CreateAction(name string) *resource.Action {
	switch name {
	case ActionGetHistory:
		return &resource.Action{
			Name: ActionGetHistory,
		}
	case ActionRollback:
		return &resource.Action{
			Name:  ActionRollback,
			Input: RollBackVersion{},
		}
	case ActionSetImage:
		return &resource.Action{
			Name:  ActionSetImage,
			Input: SetImage{},
		}
	default:
		return nil
	}
}
