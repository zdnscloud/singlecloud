package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetResourceQuotaSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type ResourceQuota struct {
	resttypes.Resource `json:",inline"`
	Name               string              `json:"name,omitempty"`
	Limits             map[string]string   `json:"limits,omitempty"`
	Status             ResourceQuotaStatus `json:"status,omitempty"`
}

type ResourceQuotaStatus struct {
	Limits map[string]string `json:"limits,omitempty"`
	Used   map[string]string `json:"used,omitempty"`
}

var ResourceQuotaType = resttypes.GetResourceType(ResourceQuota{})
