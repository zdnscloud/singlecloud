package types

import (
	"github.com/zdnscloud/gorest/types"
)

func SetNodeSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parent = "cluster"
}

type Node struct {
	types.Resource       `json:",inline"`
	Name                 string            `json:"name,omitempty"`
	Address              string            `json:"address,omitempty"`
	Role                 string            `json:"role,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	Annotations          map[string]string `json:"annotations,omitempty"`
	Status               bool              `json:"status,omitempty"`
	OperatingSystem      string            `json:"operating_system,omitempty"`
	OperatingSystemImage string            `json:"operating_system_image,omitempty"`
	DockerVersion        string            `json:"docker_version,omitempty"`
	Cpu                  string            `json:"cpu,omitempty"`
	CpuUsedRatio         string            `json:"cpu_used_ratio,omitempty"`
	Memory               string            `json:"memory,omitempty, omitempty"`
	MemoryUsedRatio      string            `json:"memory_used_ratio"`
	Storage              string            `json:"storage,omitempty"`
	StorageUserdRatio    string            `json:"storage_used_ratio, omitempty"`
	CreationTimestamp    string            `json:"creation_timestamp"`
	PodCount             int               `json:"pod_count"`
	PodUsedRatio         int               `json:"pod_used_ratio"`
}
