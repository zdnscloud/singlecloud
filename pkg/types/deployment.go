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

type PortSpec struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type Container struct {
	Name         string     `json:"name"`
	Image        string     `json:"image"`
	Command      []string   `json:"command,omitempty"`
	Args         []string   `json:"args,omitempty"`
	ConfigName   string     `json:"config_name,omitempty"`
	MountPath    string     `json:"mount_path,omitempty"`
	ExposedPorts []PortSpec `json:"exposed_ports,omitempty"`
}

type Deployment struct {
	resttypes.Resource `json:",inline"`
	Name               string      `json:"name,omitempty"`
	Replicas           uint32      `json:"replicas"`
	Containers         []Container `json:"containers"`
}

var DeploymentType = resttypes.GetResourceType(Deployment{})
