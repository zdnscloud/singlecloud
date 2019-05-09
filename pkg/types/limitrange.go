package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetLimitRangeSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type LimitRange struct {
	resttypes.Resource `json:",inline"`
	Name               string            `json:"name,omitempty"`
	Max                map[string]string `json:"max,omitempty"`
	Min                map[string]string `json:"min,omitempty"`
}

var LimitRangeType = resttypes.GetResourceType(LimitRange{})
