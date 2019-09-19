package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Config struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

//difference with k8s ConfigMap
//not support binary
type ConfigMap struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name"`
	Configs               []Config `json:"configs,omitempty"`
}

func (c ConfigMap) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
