package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type ResourceQuota struct {
	resource.ResourceBase `json:",inline"`
	Name                  string              `json:"name" rest:"required=true,isDomain=true"`
	Limits                map[string]string   `json:"limits,omitempty"`
	Status                ResourceQuotaStatus `json:"status,omitempty" rest:"description=readonly"`
}

type ResourceQuotaStatus struct {
	Limits map[string]string `json:"limits,omitempty"`
	Used   map[string]string `json:"used,omitempty"`
}

func (r ResourceQuota) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
