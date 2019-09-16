package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetSecretSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	schema.Parents = []string{NamespaceType}
}

type Secret struct {
	resttypes.Resource `json:",inline"`
	Name               string       `json:"name"`
	Data               []SecretData `json:"data"`
}

type SecretData struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var SecretType = resttypes.GetResourceType(Secret{})
