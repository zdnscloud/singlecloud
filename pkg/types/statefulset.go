package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	StorageClassNameLVM    = "lvm"
	StorageClassNameCephfs = "cephfs"
	StorageClassNameTemp   = "temporary"
)

type StatefulSet struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                     `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Replicas              int                        `json:"replicas" rest:"required=true,min=0,max=50"`
	Containers            []Container                `json:"containers" rest:"required=true"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions,omitempty" rest:"description=immutable"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes,omitempty"`
	Status                WorkloadStatus             `json:"status,omitempty" rest:"description=readonly"`
	Memo                  string                     `json:"memo,omitempty"`
}

type PersistentVolumeTemplate struct {
	Name             string `json:"name" rest:"isDomain=true"`
	Size             string `json:"size"`
	StorageClassName string `json:"storageClassName" rest:"options=lvm|cephfs|temporary"`
}

func (s StatefulSet) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (s StatefulSet) GetActions() []resource.Action {
	return []resource.Action{
		resource.Action{
			Name:   ActionGetHistory,
			Output: &VersionHistory{},
		},
		resource.Action{
			Name:  ActionRollback,
			Input: &RollBackVersion{},
		},
		resource.Action{
			Name:   ActionSetPodCount,
			Input:  &SetPodCount{},
			Output: &SetPodCount{},
		},
	}
}
