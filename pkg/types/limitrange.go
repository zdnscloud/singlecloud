package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type LimitRange struct {
	resource.ResourceBase `json:",inline"`
	Name                  string            `json:"name" rest:"required=true,isDomain=true"`
	Max                   map[string]string `json:"max,omitempty"`
	Min                   map[string]string `json:"min,omitempty"`
}

func (l LimitRange) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
