package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetNodeNetworkSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parents = []string{ClusterType}
}

type NodeNetwork struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
	IP                 string `json:"ip"`
}

var NodeNetworkType = resttypes.GetResourceType(NodeNetwork{})
