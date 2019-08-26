package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetRegistrySchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"POST", "GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parents = []string{ClusterType}
}

type Registry struct {
	resttypes.Resource `json:",inline"`
	IngressDomain      string `json:"ingressDomain"`
	StorageClass       string `json:"storageClass"`
	StorageSize        int    `json:"storageSize"`
	AdminPassword      string `json:"adminPassword"`
	RedirectUrl        string `json:"redirectUrl"`
	ApplicationLink    string `json:"applicationLink"`
}

var RegistryType = resttypes.GetResourceType(Registry{})
