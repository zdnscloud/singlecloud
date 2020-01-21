package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type UDPIngress struct {
	resource.ResourceBase `json:",inline"`
	Port                  int    `json:"port" rest:"required=true"`
	ServiceName           string `json:"serviceName" rest:"required=true,isDomain=true"`
	ServicePort           int    `json:"servicePort" rest:"required=true"`
}

func (u UDPIngress) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

/*
ing_a ---> udp/port ---> svc/port
*/
