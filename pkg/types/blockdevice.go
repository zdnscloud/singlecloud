package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type BlockDevice struct {
	resource.ResourceBase `json:",inline"`
	NodeName              string `json:"nodeName"`
	BlockDevices          []Dev  `json:"blockDevices"`
}

type Dev struct {
	Name string `json:"name"`
	Size string `json:"size"`
}

func (b BlockDevice) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
