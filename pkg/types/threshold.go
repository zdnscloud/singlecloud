package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	ThresholdTable = "threshold"
)

type Threshold struct {
	resource.ResourceBase `json:",inline"`
	Cpu                   int      `json:"cpu,omitempty" rest:"min=1,max=100"`
	Memory                int      `json:"memory,omitempty" rest:"min=1,max=100"`
	Storage               int      `json:"storage,omitempty" rest:"min=1,max=100"`
	PodCount              int      `json:"podCount,omitempty" rest:"min=1,max=100"`
	MailFrom              Mail     `json:"mailFrom,omitempty"`
	MailTo                []string `json:"mailTo,omitempty"`
}

type Mail struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
}
