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
	CurrentNumberScheduled int32                `json:"currentNumberScheduled,omitempty"`
	NumberMisscheduled     int32                `json:"numberMisscheduled,omitempty"`
	DesiredNumberScheduled int32                `json:"desiredNumberScheduled,omitempty"`
	NumberReady            int32                `json:"numberReady,omitempty"`
	ObservedGeneration     int64                `json:"observedGeneration,omitempty"`
	UpdatedNumberScheduled int32                `json:"updatedNumberScheduled,omitempty"`
	NumberAvailable        int32                `json:"numberAvailable,omitempty"`
	NumberUnavailable      int32                `json:"numberUnavailable,omitempty"`
	CollisionCount         int32                `json:"collisionCount,omitempty"`
	DaemonSetConditions    []DaemonSetCondition `json:"conditions,omitempty"`
}

type DaemonSetCondition struct {
	Type               string           `json:"type,omitempty"`
	Status             string           `json:"status,omitempty"`
	LastTransitionTime resource.ISOTime `json:"lastTransitionTime,omitempty"`
	Reason             string           `json:"reason,omitempty"`
	Message            string           `json:"message,omitempty"`
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
