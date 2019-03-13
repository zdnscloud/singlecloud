package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetIngressSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

type IngressPath struct {
	Path        string `json:"path"`
	ServiceName string `json:"serviceName"`
	ServicePort int    `json:"servicePort"`
}
type IngressRule struct {
	Host  string        `json:"host"`
	Paths []IngressPath `json:"paths"`
}

type Ingress struct {
	resttypes.Resource `json:",inline"`
	Name               string        `json:"name"`
	Rules              []IngressRule `json:"rules"`
}

var IngressType = resttypes.GetResourceType(Ingress{})
