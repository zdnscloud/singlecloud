package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type OuterService struct {
	resource.ResourceBase `json:",inline"`
	EntryPoint            string                  `json:"entryPoint"`
	Services              map[string]InnerService `json:"services"`
}

func (o OuterService) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
