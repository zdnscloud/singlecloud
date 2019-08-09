package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetServiceNetworkSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parents = []string{ClusterType}
}

type ServiceNetwork struct {
	resttypes.Resource `json:",inline"`
	Namespace          string `json:"-"`
	Name               string `json:"name"`
	IP                 string `json:"ip"`
}

var ServiceNetworkType = resttypes.GetResourceType(ServiceNetwork{})
