package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type IngressRule struct {
	Host        string `json:"host"`
	Path        string `json:"path,omitempty"`
	ServiceName string `json:"serviceName"`
	ServicePort int    `json:"servicePort"`
}

type Ingress struct {
	resource.ResourceBase `json:",inline"`
	Name                  string        `json:"name"`
	Rules                 []IngressRule `json:"rules"`
}

func (i Ingress) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

/*
ing_a ---> host1 ---> path1 --> svc/port
                      path2 --> svc/port
	  ---> host2 ---> path1 --> svc/port
*/
