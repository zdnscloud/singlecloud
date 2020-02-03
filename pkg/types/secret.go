package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Secret struct {
	resource.ResourceBase `json:",inline"`
	Name                  string       `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Data                  []SecretData `json:"data" rest:"required=true"`
}

type SecretData struct {
	Key   string `json:"key" rest:"required=true"`
	Value string `json:"value" rest:"required=true"`
}

func (s Secret) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
