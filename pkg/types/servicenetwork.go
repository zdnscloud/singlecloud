package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type ServiceNetwork struct {
	resource.ResourceBase `json:",inline"`
	Namespace             string `json:"-"`
	Name                  string `json:"name"`
	IP                    string `json:"ip"`
}

func (s ServiceNetwork) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
