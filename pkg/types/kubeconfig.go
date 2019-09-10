package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetKubeConfigSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parents = []string{ClusterType}
}

type KubeConfig struct {
	resttypes.Resource `json:",inline"`
	User               string `json:"user" rest:"required=true"`
	KubeConfig         string `json:"kubeConfig"`
}

var KubeConfigType = resttypes.GetResourceType(KubeConfig{})
