package servicecache

import (
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type Service struct {
	Name      string           `json:"name"`
	Ingress   *Ingress         `json:"ingress,omitempty"`
	Workloads []types.Workload `json:"workloads"`
}

type Ingress struct {
	Name  string        `json:"name"`
	Rules []IngressRule `json:"rules"`
}

type IngressRule struct {
	Domain string        `json:"domain,omitempty"`
	Port   int           `json:"port,omitempty"`
	Paths  []IngressPath `json:"path"`
}

type IngressPath struct {
	Service string `json:"service"`
	Path    string `json:"path"`
}
