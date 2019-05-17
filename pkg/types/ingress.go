package types

import (
	"errors"
	"strings"

	resttypes "github.com/zdnscloud/gorest/types"
)

type IngressProtocol string

var ErrUnsupportedIngressProtocol = errors.New("service protocol isn't supported")

var strToIngressProtocol = map[string]IngressProtocol{
	"UDP":  IngressProtocolUDP,
	"TCP":  IngressProtocolTCP,
	"GRPC": IngressProtocolGRPC,
	"HTTP": IngressProtocolHTTP,
}

const (
	IngressProtocolUDP  IngressProtocol = "UDP"
	IngressProtocolTCP  IngressProtocol = "TCP"
	IngressProtocolGRPC IngressProtocol = "GRPC"
	IngressProtocolHTTP IngressProtocol = "HTTP"
)

func IngressProtocolFromString(p string) (IngressProtocol, error) {
	if protocol, ok := strToIngressProtocol[strings.ToUpper(p)]; ok {
		return protocol, nil
	} else {
		return "", ErrUnsupportedIngressProtocol
	}
}

func SetIngressSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type IngressPath struct {
	Path        string `json:"path,omitempty"`
	ServiceName string `json:"serviceName"`
	ServicePort int    `json:"servicePort"`
}

type IngressRule struct {
	Host     string          `json:"host"`
	Port     int             `json:"port,omitempty"`
	Protocol IngressProtocol `json:"protocol"`
	Paths    []IngressPath   `json:"paths"`
}

type Ingress struct {
	resttypes.Resource `json:",inline"`
	Name               string        `json:"name"`
	Rules              []IngressRule `json:"rules"`
}

var IngressType = resttypes.GetResourceType(Ingress{})
