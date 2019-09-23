package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type DaemonSet struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                     `json:"name,omitempty"`
	Containers            []Container                `json:"containers,omitempty"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions,omitempty"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes"`
	Status                DaemonSetStatus            `json:"status,omitempty"`
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

type DaemonsetHistory struct {
	ControllerRevisions ControllerRevisions `json:"controllerRevisions"`
}

type ControllerRevision struct{}

type ControllerRevisions []ControllerRevision
