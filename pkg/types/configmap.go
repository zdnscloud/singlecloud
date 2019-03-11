package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetConfigMapSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

//difference with k8s ConfigMap
//not support binary
type ConfigMap struct {
	resttypes.Resource `json:",inline"`
	Name               string            `json:"name"`
	Data               map[string]string `json:"data"`
}

var ConfigMapType = resttypes.GetResourceType(ConfigMap{})
