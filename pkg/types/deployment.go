package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetDeploymentSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

type DeploymentPort struct {
	Name              string `json:"name"`
	Port              int    `json:"port"`
	Protocol          string `json:"protocol"`
	ServiceType       string `json:"serviceType"`
	ServicePort       int    `json:"servicePort"`
	IngressDomainName string `json:"ingressDomainName"`
}

type Container struct {
	Name         string           `json:"name"`
	Image        string           `json:"image"`
	Command      []string         `json:"command,omitempty"`
	Args         []string         `json:"args,omitempty"`
	ConfigName   string           `json:"configName,omitempty"`
	MountPath    string           `json:"mountPath,omitempty"`
	ExposedPorts []DeploymentPort `json:"exposedPorts,omitempty"`
}

type Deployment struct {
	resttypes.Resource `json:",inline"`
	Name               string      `json:"name,omitempty"`
	Replicas           int         `json:"replicas"`
	Containers         []Container `json:"containers"`
}

var DeploymentType = resttypes.GetResourceType(Deployment{})
