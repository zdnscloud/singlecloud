package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetResourceQuotaSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

type ResourceQuota struct {
	resttypes.Resource `json:",inline"`
	Name               string              `json:"name,omitempty"`
	Hard               map[string]string   `json:"hard,omitempty"`
	Scopes             []string            `json:"scopes,omitempty"`
	ScopeSelectors     []ScopeSelector     `json:"scopeSelectors,omitempty"`
	Status             ResourceQuotaStatus `json:"status,omitempty"`
}

type ScopeSelector struct {
	ScopeName string   `json:"scopeName,omitempty"`
	Operator  string   `json:"operator,omitempty"`
	Values    []string `json:"values,omitempty"`
}

type ResourceQuotaStatus struct {
	Hard map[string]string `json:"hard,omitempty"`
	Used map[string]string `json:"used,omitempty"`
}

var ResourceQuotaType = resttypes.GetResourceType(ResourceQuota{})
