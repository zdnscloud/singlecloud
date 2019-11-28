package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Job struct {
	resource.ResourceBase `json:",inline"`
	Name                  string      `json:"name,omitempty"`
	RestartPolicy         string      `json:"restartPolicy,omitempty"`
	Containers            []Container `json:"containers"`
	Status                JobStatus   `json:"status" rest:"description=readonly"`
}

type JobStatus struct {
	StartTime      resource.ISOTime `json:"startTime,omitempty" rest:"description=readonly"`
	CompletionTime resource.ISOTime `json:"completionTime,omitempty" rest:"description=readonly"`
	Active         int32            `json:"active,omitempty" rest:"description=readonly"`
	Succeeded      int32            `json:"succeeded,omitempty" rest:"description=readonly"`
	Failed         int32            `json:"failed,omitempty" rest:"description=readonly"`
	JobConditions  []JobCondition   `json:"jobConditions,omitempty" rest:"description=readonly"`
}

type JobCondition struct {
	Type               string           `json:"type,omitempty" rest:"description=readonly"`
	Status             string           `json:"status,omitempty" rest:"description=readonly"`
	LastProbeTime      resource.ISOTime `json:"lastProbeTime,omitempty" rest:"description=readonly"`
	LastTransitionTime resource.ISOTime `json:"lastTransitionTime,omitempty" rest:"description=readonly"`
	Reason             string           `json:"reason,omitempty" rest:"description=readonly"`
	Message            string           `json:"message,omitempty" rest:"description=readonly"`
}

func (j Job) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
