package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetRegistryCertSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET", "PUT"}
}

type RegistryCert struct {
	resttypes.Resource `json:",inline"`
	Domain             string                      `json:"domain"`
	Cert               string                      `json:"cert"`
	Clusters           map[string]RegistryCertNode `json:"clusters"`
}

type RegistryCertNode struct {
	Name       string `json:"name"`
	isDeployed bool   `json:"isDeployed"`
}

var RegistryCertType = resttypes.GetResourceType(RegistryCert{})
