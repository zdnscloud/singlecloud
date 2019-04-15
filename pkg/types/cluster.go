package types

import (
	"github.com/zdnscloud/gorest/types"
)

func SetClusterSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET"}
}

type Cluster struct {
	types.Resource `json:",inline"`
	Name           string `json:"name,omitempty"`
	NodesCount     int    `json:"nodeCount,omitempty"`
	Version        string `json:"version,omitempty"`
}

var ClusterType = types.GetResourceType(Cluster{})
