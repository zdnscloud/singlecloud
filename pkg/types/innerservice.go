package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type InnerService struct {
	resource.ResourceBase `json:",inline"`
	Name                  string      `json:"name"`
	Workloads             []*Workload `json:"workloads"`
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

func (i InnerService) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
