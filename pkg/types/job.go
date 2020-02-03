package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Job struct {
	resource.ResourceBase `json:",inline"`
	Name                  string      `json:"name" rest:"required=true,isDomain=true"`
	RestartPolicy         string      `json:"restartPolicy" rest:"required=true,options=OnFailure|Never"`
	Containers            []Container `json:"containers" rest:"required=true"`
	Status                JobStatus   `json:"status,omitempty" rest:"description=readonly"`
}

type JobStatus struct {
	StartTime      resource.ISOTime `json:"startTime,omitempty"`
	CompletionTime resource.ISOTime `json:"completionTime,omitempty"`
	Active         int32            `json:"active,omitempty"`
	Succeeded      int32            `json:"succeeded,omitempty"`
	Failed         int32            `json:"failed,omitempty"`
	JobConditions  []JobCondition   `json:"jobConditions,omitempty"`
}

type JobCondition struct {
	Type               string           `json:"type,omitempty"`
	Status             string           `json:"status,omitempty"`
	LastProbeTime      resource.ISOTime `json:"lastProbeTime,omitempty"`
	LastTransitionTime resource.ISOTime `json:"lastTransitionTime,omitempty"`
	Reason             string           `json:"reason,omitempty"`
	Message            string           `json:"message,omitempty"`
}

func (j Job) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
