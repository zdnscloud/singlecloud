package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type NamespaceThreshold struct {
	resource.ResourceBase `json:",inline"`
	Cpu                   int      `json:"cpu,omitempty" rest:"min=0,max=99"`
	Memory                int      `json:"memory,omitempty" rest:"min=0,max=99"`
	Storage               int      `json:"storage,omitempty" rest:"min=0,max=99"`
	PodStorage            int      `json:"podStorage,omitempty" rest:"min=0,max=99"`
	MailTo                []string `json:"mailTo,omitempty"`
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
