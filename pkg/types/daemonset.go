package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetDaemonSetSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE", "POST"}
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

type DaemonSet struct {
	resttypes.Resource `json:",inline"`
	Name               string                     `json:"name,omitempty"`
	Containers         []Container                `json:"containers,omitempty"`
	AdvancedOptions    AdvancedOptions            `json:"advancedOptions,omitempty"`
	PersistentVolumes  []PersistentVolumeTemplate `json:"persistentVolumes"`
	Status             DaemonSetStatus            `json:"status,omitempty"`
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

var DaemonSetType = resttypes.GetResourceType(DaemonSet{})

type DaemonsetHistory struct {
	ControllerRevisions ControllerRevisions `json:"controllerRevisions"`
}

type ControllerRevision struct{}

type ControllerRevisions []ControllerRevision
