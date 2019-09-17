package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetUDPIngressSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type UdpIngress struct {
	resttypes.Resource `json:",inline"`
	Port               int    `json:"port,omitempty"`
	ServiceName        string `json:"serviceName"`
	ServicePort        int    `json:"servicePort"`
}

var UdpIngressType = resttypes.GetResourceType(UdpIngress{})

/*
ing_a ---> udp/port ---> svc/port
*/
