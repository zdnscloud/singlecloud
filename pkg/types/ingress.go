package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetIngressSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type IngressRule struct {
	Host        string `json:"host"`
	Path        string `json:"path,omitempty"`
	ServiceName string `json:"serviceName"`
	ServicePort int    `json:"servicePort"`
}

type Ingress struct {
	resttypes.Resource `json:",inline"`
	Name               string        `json:"name"`
	Rules              []IngressRule `json:"rules"`
}

var IngressType = resttypes.GetResourceType(Ingress{})

/*
ing_a ---> host1 ---> path1 --> svc/port
                      path2 --> svc/port
	  ---> host2 ---> path1 --> svc/port
*/
