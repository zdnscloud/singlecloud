package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetNodeSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parent = ClusterType
}

type Node struct {
	resttypes.Resource   `json:",inline"`
	Name                 string            `json:"name,omitempty"`
	Address              string            `json:"address,omitempty"`
	Role                 string            `json:"role,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	Annotations          map[string]string `json:"annotations,omitempty"`
	OperatingSystem      string            `json:"operatingSystem,omitempty"`
	OperatingSystemImage string            `json:"operatingSystemImage,omitempty"`
	DockerVersion        string            `json:"dockerVersion,omitempty"`
	Cpu                  int64             `json:"cpu"`
	CpuUsed              int64             `json:"cpuUsed"`
	CpuUsedRatio         string            `json:"cpuUsedRatio"`
	Memory               int64             `json:"memory"`
	MemoryUsed           int64             `json:"memoryUsed"`
	MemoryUsedRatio      string            `json:"memoryUsedRatio"`
	Pod                  int64             `json:"pod"`
	PodUsed              int64             `json:"podUsed"`
	PodUsedRatio         string            `json:"podUsedRatio"`
}

var NodeType = resttypes.GetResourceType(Node{})
