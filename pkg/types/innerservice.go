package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetInnerServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parents = []string{NamespaceType}
}

type InnerService struct {
	resttypes.Resource `json:",inline"`
	Name               string      `json:"name"`
	Workloads          []*Workload `json:"workloads"`
}

type Workload struct {
	Name string        `json:"name"`
	Kind string        `json:"kind"`
	Pods []WorkloadPod `json:"pods"`
}

type WorkloadPod struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

var InnerServiceType = resttypes.GetResourceType(InnerService{})
