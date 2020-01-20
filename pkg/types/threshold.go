package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	DefaultCpu                        = 80
	DefaultMemory                     = 80
	DefaultStorage                    = 80
	DefaultPodCount                   = 80
	ThresholdActive   ThresholdStatus = "active"
	ThresholdInActive ThresholdStatus = "inactive"
)

type Threshold struct {
	resource.ResourceBase `json:",inline"`
	Cpu                   int             `json:"cpu,omitempty" rest:"min=1,max=100"`
	Memory                int             `json:"memory,omitempty" rest:"min=1,max=100"`
	Storage               int             `json:"storage,omitempty" rest:"min=1,max=100"`
	PodCount              int             `json:"podCount,omitempty" rest:"min=1,max=100"`
	MailFrom              Mail            `json:"mailFrom,omitempty"`
	MailTo                []string        `json:"mailTo,omitempty"`
	Status                ThresholdStatus `json:"status" rest:"description=readonly"`
}

type ThresholdStatus string

type Mail struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     string `json:"port,omitempty"`
}

func (t Threshold) CreateDefaultResource() resource.Resource {
	return &Threshold{
		Cpu:      DefaultCpu,
		Memory:   DefaultMemory,
		Storage:  DefaultStorage,
		PodCount: DefaultPodCount,
	}
}
