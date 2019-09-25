package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type NodeNetwork struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name"`
	IP                    string `json:"ip"`
}

func (n NodeNetwork) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
