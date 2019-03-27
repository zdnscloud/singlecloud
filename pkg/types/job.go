package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetJobSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

type Job struct {
	resttypes.Resource `json:",inline"`
	Name               string      `json:"name,omitempty"`
	Parallelism        int32       `json:"parallelism,omitempty"`
	Completions        int32       `json:"completions,omitempty"`
	BackoffLimit       int32       `json:"backoffLimit,omitempty"`
	NodeName           string      `json:"nodeName,omitempty"`
	Containers         []Container `json:"containers,omitempty"`
	Status             JobStatus   `json:"status,omitempty"`
}

type JobStatus struct {
	StartTime      resttypes.ISOTime `json:"startTime,omitempty"`
	CompletionTime resttypes.ISOTime `json:"completionTime,omitempty"`
	Active         int32             `json:"active,omitempty"`
	Succeeded      int32             `json:"succeeded,omitempty"`
	Failed         int32             `json:"failed,omitempty"`
	JobConditions  []JobCondition    `json:"jobConditions,omitempty"`
}

type JobCondition struct {
	Type               string            `json:"type,omitempty"`
	Status             string            `json:"status,omitempty"`
	LastProbeTime      resttypes.ISOTime `json:"lastProbeTime,omitempty"`
	LastTransitionTime resttypes.ISOTime `json:"lastTransitionTime,omitempty"`
	Reason             string            `json:"reason,omitempty"`
	Message            string            `json:"message,omitempty"`
}

var JobType = resttypes.GetResourceType(Job{})
