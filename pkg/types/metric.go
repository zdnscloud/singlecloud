package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Metric struct {
	resource.ResourceBase `json:",inline"`
	Name                  string         `json:"name,omitempty"`
	Type                  string         `json:"type,omitempty"`
	Help                  string         `json:"help,omitempty"`
	Metrics               []MetricFamily `json:"metrics,omitempty"`
}

type MetricFamily struct {
	Labels  map[string]string `json:"labels,omitempty"`
	Gauge   Gauge             `json:"gauge,omitempty"`
	Counter Counter           `json:"counter,omitempty"`
}

type Gauge struct {
	Value int `json:"value,omitempty"`
}

type Counter struct {
	Value int `json:"value,omitempty"`
}

func (m Metric) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Deployment{}, DaemonSet{}, StatefulSet{}}
}
