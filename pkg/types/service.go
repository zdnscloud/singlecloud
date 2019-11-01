package types

import (
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/zdnscloud/gorest/resource"
)

type ServicePort struct {
	Name       string             `json:"name"`
	Port       int                `json:"port"`
	TargetPort intstr.IntOrString `json:"targetPort"`
	Protocol   string             `json:"protocol"`
	NodePort   int                `json:"nodePort,omitempty"`
}

type Service struct {
	resource.ResourceBase `json:",inline"`
	Name                  string        `json:"name"`
	ServiceType           string        `json:"serviceType"`
	Headless              bool          `json:"headless"`
	ClusterIP             string        `json:"clusterIP,omitempty"`
	ExposedPorts          []ServicePort `json:"exposedPorts,omitempty"`
}

func (s Service) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
