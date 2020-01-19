package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type DaemonSet struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                     `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Replicas              int                        `json:"replicas" rest:"description=readonly"`
	Containers            []Container                `json:"containers" rest:"required=true"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions,omitempty" rest:"description=immutable"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes,omitempty"`
	Status                WorkloadStatus             `json:"status,omitempty" rest:"description=readonly"`
	Memo                  string                     `json:"memo,omitempty"`
}

func (d DaemonSet) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

var DaemonSetActions = []resource.Action{
	resource.Action{
		Name:   ActionGetHistory,
		Output: &VersionHistory{},
	},
	resource.Action{
		Name:  ActionRollback,
		Input: &RollBackVersion{},
	},
}

func (d DaemonSet) GetActions() []resource.Action {
	return DaemonSetActions
}
