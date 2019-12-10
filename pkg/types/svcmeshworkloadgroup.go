package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type SvcMeshWorkloadGroup struct {
	resource.ResourceBase `json:",inline"`
	Workloads             SvcMeshWorkloads `json:"workloads,omitempty"`
}

func (w SvcMeshWorkloadGroup) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

type SvcMeshWorkloadGroups []*SvcMeshWorkloadGroup
