package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type DaemonSet struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                     `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Containers            []Container                `json:"containers" rest:"required=true"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions,omitempty" rest:"description=immutable"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes,omitempty"`
	Status                DaemonSetStatus            `json:"status,omitempty" rest:"description=readonly"`
	Memo                  string                     `json:"memo,omitempty"`
}

type DaemonSetStatus struct {
	CurrentNumberScheduled int                 `json:"currentNumberScheduled,omitempty"`
	NumberMisscheduled     int                 `json:"numberMisscheduled,omitempty"`
	DesiredNumberScheduled int                 `json:"desiredNumberScheduled,omitempty"`
	NumberReady            int                 `json:"numberReady,omitempty"`
	ObservedGeneration     int                 `json:"observedGeneration,omitempty"`
	UpdatedNumberScheduled int                 `json:"updatedNumberScheduled,omitempty"`
	NumberAvailable        int                 `json:"numberAvailable,omitempty"`
	NumberUnavailable      int                 `json:"numberUnavailable,omitempty"`
	CollisionCount         int                 `json:"collisionCount,omitempty"`
	Conditions             []WorkloadCondition `json:"conditions,omitempty"`
}

func (d DaemonSet) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (d DaemonSet) CreateAction(name string) *resource.Action {
	switch name {
	case ActionGetHistory:
		return &resource.Action{
			Name: ActionGetHistory,
		}
	case ActionRollback:
		return &resource.Action{
			Name:  ActionRollback,
			Input: &RollBackVersion{},
		}
	default:
		return nil
	}
}
