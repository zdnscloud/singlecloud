package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type BlockDevice struct {
	resource.ResourceBase `json:",inline"`
	NodeName              string `json:"nodeName"`
	BlockDevices          []Dev  `json:"blockDevices"`
	UsedBy                string `json:"usedby"`
}

type Dev struct {
	Name   string `json:"name"`
	Size   string `json:"size"`
	UsedBy string `json:"-"`
}

func (b BlockDevice) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

type ClusterAgentBlockDevice struct {
	resource.ResourceBase `json:",inline"`
        NodeName           string `json:"nodeName"`
        BlockDevices       []ClusterAgentDev  `json:"blockDevices"`
}
type ClusterAgentDev struct {
        Name       string `json:"name"`
        Size       string `json:"size"`
        Parted     bool   `json:"parted"`
        Filesystem bool   `json:"filesystem"`
        Mount      bool   `json:"mount"`
}

