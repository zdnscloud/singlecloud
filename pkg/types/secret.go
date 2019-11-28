package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Secret struct {
	resource.ResourceBase `json:",inline"`
	Name                  string       `json:"name" rest:"description=immutable"`
	Data                  []SecretData `json:"data"`
}

type SecretData struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s Secret) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
