package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type NamespaceThreshold struct {
	resource.ResourceBase `json:",inline"`
	Cpu                   int `json:"cpu,omitempty" rest:"min=0,max=100"`
	Memory                int `json:"memory,omitempty" rest:"min=0,max=100"`
	Storage               int `json:"storage,omitempty" rest:"min=0,max=100"`
	PodStorage            int `json:"podStorage,omitempty" rest:"min=0,max=100"`
}

func (t NamespaceThreshold) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (t NamespaceThreshold) CreateDefaultResource() resource.Resource {
	return &NamespaceThreshold{
		Cpu:        DefaultCpu,
		Memory:     DefaultMemory,
		Storage:    DefaultStorage,
		PodStorage: DefaultPodStorage,
	}
}
