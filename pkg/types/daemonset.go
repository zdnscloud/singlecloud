package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type DaemonSet struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                     `json:"name,omitempty" rest:"description=immutable"`
	Containers            []Container                `json:"containers,omitempty"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions,omitempty" rest:"description=immutable"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes"`
	Status                DaemonSetStatus            `json:"status,omitempty" rest:"description=readonly"`
	Memo                  string                     `json:"memo,omitempty"`
}

type DaemonSetStatus struct {
	CurrentNumberScheduled int                 `json:"currentNumberScheduled,omitempty" rest:"description=readonly"`
	NumberMisscheduled     int                 `json:"numberMisscheduled,omitempty" rest:"description=readonly"`
	DesiredNumberScheduled int                 `json:"desiredNumberScheduled,omitempty" rest:"description=readonly"`
	NumberReady            int                 `json:"numberReady,omitempty" rest:"description=readonly"`
	ObservedGeneration     int                 `json:"observedGeneration,omitempty" rest:"description=readonly"`
	UpdatedNumberScheduled int                 `json:"updatedNumberScheduled,omitempty" rest:"description=readonly"`
	NumberAvailable        int                 `json:"numberAvailable,omitempty" rest:"description=readonly"`
	NumberUnavailable      int                 `json:"numberUnavailable,omitempty" rest:"description=readonly"`
	CollisionCount         int                 `json:"collisionCount,omitempty" rest:"description=readonly"`
	Conditions             []WorkloadCondition `json:"conditions,omitempty" rest:"description=readonly"`
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
