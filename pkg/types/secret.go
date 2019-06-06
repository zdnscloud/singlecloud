package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetSecretSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type Secret struct {
	resttypes.Resource `json:",inline"`
	Name               string            `json:"name"`
	Data               map[string]string `json:"data"`
}

var SecretType = resttypes.GetResourceType(Secret{})
