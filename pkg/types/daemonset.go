package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetDaemonSetSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

type DaemonSet struct {
	resttypes.Resource `json:",inline"`
	Name               string          `json:"name,omitempty"`
	Containers         []Container     `json:"containers,omitempty"`
	AdvancedOptions    AdvancedOptions `json:"advancedOptions,omitempty"`
	Status             DaemonSetStatus `json:"status,omitempty"`
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
	Type               string            `json:"type,omitempty"`
	Status             string            `json:"status,omitempty"`
	LastTransitionTime resttypes.ISOTime `json:"lastTransitionTime,omitempty"`
	Reason             string            `json:"reason,omitempty"`
	Message            string            `json:"message,omitempty"`
}

var DaemonSetType = resttypes.GetResourceType(DaemonSet{})
