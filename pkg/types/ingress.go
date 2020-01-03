package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type IngressRule struct {
	Host        string `json:"host" rest:"required=true,isDomain=true"`
	Path        string `json:"path" rest:"required=true"`
	ServiceName string `json:"serviceName" rest:"required=true,isDomain=true"`
	ServicePort int    `json:"servicePort" rest:"required=true"`
}

type Ingress struct {
	resource.ResourceBase `json:",inline"`
	Name                  string        `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Rules                 []IngressRule `json:"rules" rest:"required=true"`
	MaxBodySize           int           `json:"maxBodySize"`
	MaxBodySizeUnit       string        `json:"maxBodySizeUnit" rest:"options=m|k"`
}

func (i Ingress) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

/*
ing_a ---> host1 ---> path1 --> svc/port
                      path2 --> svc/port
	  ---> host2 ---> path1 --> svc/port
*/
