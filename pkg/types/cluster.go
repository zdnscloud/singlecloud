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

	Cpu             int64  `json:"cpu,omitempty"`
	CpuUsed         int64  `json:"-"`
	CpuUsedRatio    string `json:"cpuUsedRatio,omitempty"`
	Memory          int64  `json:"memory,omitempty"`
	MemoryUsed      int64  `json:"-"`
	MemoryUsedRatio string `json:"memoryUsedRatio,omitempty"`
	Pod             int64  `json:"pod"`
	PodUsed         int64  `json:"-"`
	PodUsedRatio    string `json:"podUsedRatio,omitempty"`
}

var ClusterType = types.GetResourceType(Cluster{})
