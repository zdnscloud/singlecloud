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
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type Container struct {
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	Command      []string          `json:"command,omitempty"`
	Args         []string          `json:"args,omitempty"`
	ConfigName   string            `json:"configName,omitempty"`
	MountPath    string            `json:"mountPath,omitempty"`
	ExposedPorts []DeploymentPort  `json:"exposedPorts,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	SecretName   string            `json:"secretName,omitempty"`
	SecretPath   string            `json:"secretPath,omitempty"`
}

type ExposedService struct {
	Name              string `json:"name"`
	Port              int    `json:"port"`
	Protocol          string `json:"protocol"`
	ServicePort       int    `json:"servicePort"`
	AutoCreateIngress bool   `json:"autoCreateIngress"`
	IngressDomainName string `json:"ingressDomainName"`
	IngressPath       string `json:"ingressPath"`
}

type AdvancedOptions struct {
	ExposedServiceType string           `json:"exposedServiceType"`
	ExposedServices    []ExposedService `json:"exposedServices"`
}

type Deployment struct {
	resttypes.Resource `json:",inline"`
	Name               string          `json:"name,omitempty"`
	Replicas           int             `json:"replicas"`
	Containers         []Container     `json:"containers"`
	AdvancedOptions    AdvancedOptions `json:"advancedOptions"`
}

var DeploymentType = resttypes.GetResourceType(Deployment{})
