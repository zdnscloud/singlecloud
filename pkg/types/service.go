package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

type ServicePort struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	TargetPort int    `json:"targetPort"`
	Protocol   string `json:"protocol"`
}

type Service struct {
	resttypes.Resource `json:",inline"`
	Name               string        `json:"name"`
	ServiceType        string        `json:"serviceType"`
	ExposedPorts       []ServicePort `json:"exposedPorts,omitempty"`
}

var ServiceType = resttypes.GetResourceType(Service{})
