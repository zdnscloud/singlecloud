package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetLimitRangeSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

type LimitRange struct {
	resttypes.Resource `json:",inline"`
	Name               string           `json:"name,omitempty"`
	Limits             []LimitRangeItem `json:"limits,omitempty"`
}

type LimitRangeItem struct {
	Type                 string            `json:"type,omitempty"`
	Max                  map[string]string `json:"max,omitempty"`
	Min                  map[string]string `json:"min,omitempty"`
	Default              map[string]string `json:"resourceList,omitempty"`
	DefaultRequest       map[string]string `json:"defaultRequest,omitempty"`
	MaxLimitRequestRatio map[string]string `json:"maxLimitRequestRatio,omitempty"`
}

var LimitRangeType = resttypes.GetResourceType(LimitRange{})
