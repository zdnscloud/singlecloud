package types

import (
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/charts"
)

type Chart struct {
	resource.ResourceBase `json:",inline"`
	Name                  string         `json:"name" rest:"description=readonly"`
	Description           string         `json:"description" rest:"description=readonly"`
	Icon                  string         `json:"icon" rest:"description=readonly"`
	Versions              []ChartVersion `json:"versions" rest:"description=readonly"`
}

type ChartVersion struct {
	Version string              `json:"version" rest:"description=readonly"`
	Config  charts.ChartConfigs `json:"config,omitempty" rest:"description=readonly"`
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

func (c Chart) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
