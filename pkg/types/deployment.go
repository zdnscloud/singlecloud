package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	ActionGetHistory = "history"
	ActionRollback   = "rollback"
	ActionSetImage   = "setImage"
)

func SetDeploymentSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE", "POST"}
	schema.Parents = []string{NamespaceType}
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name: ActionGetHistory,
	})
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionRollback,
		Input: RollBackVersion{},
	})
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionSetImage,
		Input: SetImage{},
	})
}

type ContainerPort struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type Container struct {
	Name         string          `json:"name"`
	Image        string          `json:"image"`
	Command      []string        `json:"command,omitempty"`
	Args         []string        `json:"args,omitempty"`
	ConfigName   string          `json:"configName,omitempty"`
	MountPath    string          `json:"mountPath,omitempty"`
	ExposedPorts []ContainerPort `json:"exposedPorts,omitempty"`
	Env          []EnvVar        `json:"env,omitempty"`
	SecretName   string          `json:"secretName,omitempty"`
	SecretPath   string          `json:"secretPath,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type ExposedService struct {
	ContainerPortName string          `json:"containerPortName"`
	ServicePort       int             `json:"servicePort"`
	AutoCreateIngress bool            `json:"autoCreateIngress"`
	IngressProtocol   IngressProtocol `json:"ingressProtocol"`
	IngressHost       string          `json:"ingressHost,omitempty"`
	IngressPath       string          `json:"ingressPath,omitempty"`
	IngressPort       int             `json:"ingressPort,omitempty"`
}

type AdvancedOptions struct {
	ExposedServiceType string           `json:"exposedServiceType"`
	ExposedServices    []ExposedService `json:"exposedServices"`
	ExposedMetric      ExposedMetric    `json:"exposedMetric"`
}

type Deployment struct {
	resttypes.Resource `json:",inline"`
	Name               string          `json:"name,omitempty"`
	Replicas           int             `json:"replicas"`
	Containers         []Container     `json:"containers"`
	AdvancedOptions    AdvancedOptions `json:"advancedOptions"`
}

type ExposedMetric struct {
	Path string `json:"path"`
	Port int    `json:"port"`
}

var DeploymentType = resttypes.GetResourceType(Deployment{})

type VersionHistory struct {
	VersionInfos VersionInfos `json:"history"`
}

type VersionInfo struct {
	Name         string      `json:"name"`
	Namespace    string      `json:"namespace"`
	Version      int         `json:"version"`
	ChangeReason string      `json:"changeReason"`
	Containers   []Container `json:"containers"`
}

type VersionInfos []VersionInfo

func (vs VersionInfos) Len() int {
	return len(vs)
}
func (vs VersionInfos) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}
func (vs VersionInfos) Less(i, j int) bool {
	return vs[i].Version < vs[j].Version
}

type RollBackVersion struct {
	Version int    `json:"version"`
	Reason  string `json:"reason"`
}

type SetImage struct {
	Reason string           `json:"reason"`
	Images []ContainerImage `json:"images"`
}

type ContainerImage struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}
