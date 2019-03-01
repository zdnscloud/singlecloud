package node

import (
	"github.com/zdnscloud/gorest/types"
)

type Node struct {
	ID                   string                 `json:"id,omitempty"`
	Type                 string                 `json:"type,omitempty"`
	Name                 string                 `json:"name,omitempty"`
	Address              string                 `json:"address,omitempty"`
	Role                 []string               `json:"role,omitempty"`
	Labels               map[string]interface{} `json:"labels,omitempty"`
	Annotations          map[string]interface{} `json:"annotations,omitempty"`
	Status               bool                   `json:"status,omitempty"`
	OperatingSystem      string                 `json:"operating_system,omitempty"`
	OperatingSystemImage string                 `json:"operating_system_image,omitempty"`
	DockerVersion        string                 `json:"docker_version,omitempty"`
	Cpu                  uint32                 `json:"cpu,omitempty"`
	CpuUsedRatio         string                 `json:"cpu_used_ratio,omitempty"`
	Memory               string                 `json:"memory,omitempty"`
	MemoryUsedRatio      string                 `json:"memory_used_ratio"`
	CreationTimestamp    string                 `json:"creation_timestamp"`
	Parent               types.Parent           `json:"-"`
}

func (n *Node) GetID() string {
	return n.ID
}

func (n *Node) SetID(id string) {
	n.ID = id
}

func (n *Node) GetType() string {
	return n.Type
}

func (n *Node) SetType(typ string) {
	n.Type = typ
}

func (n *Node) GetParent() types.Parent {
	return n.Parent
}

func (n *Node) SetParent(parent types.Parent) {
	n.Parent = parent
}
