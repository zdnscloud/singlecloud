package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type UdpIngress struct {
	resource.ResourceBase `json:",inline"`
	Port                  int    `json:"port,omitempty"`
	ServiceName           string `json:"serviceName"`
	ServicePort           int    `json:"servicePort"`
}

func (u UdpIngress) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

/*
ing_a ---> udp/port ---> svc/port
*/
