package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	DefaultCpu        = 80
	DefaultMemory     = 80
	DefaultStorage    = 80
	DefaultPodCount   = 80
	DefaultNodeCpu    = 80
	DefaultNodeMemory = 80
	DefaultPodStorage = 80
)

type ClusterThreshold struct {
	resource.ResourceBase `json:",inline"`
	Cpu                   int `json:"cpu,omitempty" rest:"min=0,max=100"`
	Memory                int `json:"memory,omitempty" rest:"min=0,max=100"`
	Storage               int `json:"storage,omitempty" rest:"min=0,max=100"`
	PodCount              int `json:"podCount,omitempty" rest:"min=0,max=100"`
	NodeCpu               int `json:"nodeCpu,omitempty" rest:"min=0,max=100"`
	NodeMemory            int `json:"nodeMemory,omitempty" rest:"min=0,max=100"`
}

func (t ClusterThreshold) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (t ClusterThreshold) CreateDefaultResource() resource.Resource {
	return &ClusterThreshold{
		Cpu:        DefaultCpu,
		Memory:     DefaultMemory,
		Storage:    DefaultStorage,
		PodCount:   DefaultPodCount,
		NodeCpu:    DefaultNodeCpu,
		NodeMemory: DefaultNodeMemory,
	}
}
