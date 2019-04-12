package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetInnerServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parent = NamespaceType
}

type InnerService struct {
	resttypes.Resource `json:",inline"`
	Name               string     `json:"name"`
	Workloads          []Workload `json:"workloads"`
}

type Workload struct {
	Name string      `json:"name"`
	Kind string      `json:"kind"`
	Pods []SimplePod `json:"pods"`
}

type SimplePod struct {
	Name    string `json:"name"`
	IsReady bool   `json:"isReady"`
}

var InnerServiceType = resttypes.GetResourceType(InnerService{})

type OuterService struct {
	resttypes.Resource `json:",inline"`
	Domain             string                  `json:"domain"`
	Port               int                     `json:"port"`
	Services           map[string]InnerService `json:"services"`
}

func SetOuterServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parent = NamespaceType
}

var OuterServiceType = resttypes.GetResourceType(OuterService{})
