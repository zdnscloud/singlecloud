package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type PodNetwork struct {
	resource.ResourceBase `json:",inline"`
	NodeName              string  `json:"nodeName"`
	PodCIDR               string  `json:"podCIDR"`
	PodIPs                []PodIP `json:"podIPs"`
}

type PodIP struct {
	Namespace string `json:"-"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
}

func (p PodNetwork) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
