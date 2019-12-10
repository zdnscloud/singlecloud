package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type SvcMeshWorkload struct {
	resource.ResourceBase `json:",inline"`
	Destinations          []string    `json:"destinations,omitempty"`
	Stat                  Stat        `json:"stat,omitempty"`
	Inbound               Stats       `json:"inbound,omitempty"`
	Outbound              Stats       `json:"outbound,omitempty"`
	Pods                  SvcMeshPods `json:"pods,omitempty"`
}

func (w SvcMeshWorkload) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{SvcMeshWorkloadGroup{}}
}

type SvcMeshWorkloads []*SvcMeshWorkload
