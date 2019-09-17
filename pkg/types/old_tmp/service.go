package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type ServicePort struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	TargetPort int    `json:"targetPort"`
	Protocol   string `json:"protocol"`
	NodePort   int    `json:"nodePort,omitempty"`
}

type Service struct {
	resttypes.Resource `json:",inline"`
	Name               string        `json:"name"`
	ServiceType        string        `json:"serviceType"`
	Headless           bool          `json:"headless"`
	ClusterIP          string        `json:"clusterIP,omitempty"`
	ExposedPorts       []ServicePort `json:"exposedPorts,omitempty"`
}

var ServiceType = resttypes.GetResourceType(Service{})
