package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetChartSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
}

type Chart struct {
	resttypes.Resource `json:",inline"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Icon               string         `json:"icon"`
	Versions           []ChartVersion `json:"versions"`
}

type ChartVersion struct {
	Version string `json:"version"`
	Config  string `json:"config,omitempty"`
}

type Charts []*Chart

func (c Charts) Len() int {
	return len(c)
}

func (c Charts) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Charts) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

var ChartType = resttypes.GetResourceType(Chart{})
