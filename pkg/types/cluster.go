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

	Cpu             float64 `json:"cpu,omitempty"`
	CpuUsed         float64 `json:"-"`
	CpuUsedRatio    string  `json:"cpuUsedRatio,omitempty"`
	Memory          float64 `json:"memory,omitempty"`
	MemoryUsed      float64 `json:"-"`
	MemoryUsedRatio string  `json:"memoryUsedRatio,omitempty"`
	Pod             float64 `json:"pod"`
	PodUsed         float64 `json:"-"`
	PodUsedRatio    string  `json:"podUsedRatio,omitempty"`
}

var ClusterType = types.GetResourceType(Cluster{})
