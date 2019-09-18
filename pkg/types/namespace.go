package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Namespace struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name,omitempty"`
}

func (n Namespace) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
