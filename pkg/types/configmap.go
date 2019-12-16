package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Config struct {
	Name string `json:"name" rest:"required=true,isDomain=true"`
	Data string `json:"data" rest:"required=true"`
}

//difference with k8s ConfigMap
//not support binary
type ConfigMap struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Configs               []Config `json:"configs" rest:"required=true"`
}

func (c ConfigMap) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
