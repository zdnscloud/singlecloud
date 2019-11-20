package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type UDPIngress struct {
	resource.ResourceBase `json:",inline"`
	Port                  int    `json:"port,omitempty"`
	ServiceName           string `json:"serviceName"`
	ServicePort           int    `json:"servicePort"`
}

func (u UDPIngress) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

/*
ing_a ---> udp/port ---> svc/port
*/
