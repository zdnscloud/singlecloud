package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type NodeStatus string

const (
	NSReady    NodeStatus = "Ready"
	NSNotReady NodeStatus = "NotReady"
)

type NodeRole string

const (
	RoleControlPlane NodeRole = "controlplane"
	RoleEtcd         NodeRole = "etcd"
	RoleWorker       NodeRole = "worker"
	RoleEdge         NodeRole = "edge"
	RoleStorage      NodeRole = "storage"
)

type Node struct {
	resource.ResourceBase `json:",inline"`
	Name                  string            `json:"name" rest:"required=true,minLen=1,maxLen=128"`
	Status                NodeStatus        `json:"status"`
	Address               string            `json:"address,omitempty" rest:"required=true,minLen=1,maxLen=128"`
	Roles                 []NodeRole        `json:"roles,omitempty" rest:"required=true"`
	Labels                map[string]string `json:"labels,omitempty"`
	Annotations           map[string]string `json:"annotations,omitempty"`
	OperatingSystem       string            `json:"operatingSystem,omitempty"`
	OperatingSystemImage  string            `json:"operatingSystemImage,omitempty"`
	DockerVersion         string            `json:"dockerVersion,omitempty"`
	Cpu                   int64             `json:"cpu"`
	CpuUsed               int64             `json:"cpuUsed"`
	CpuUsedRatio          string            `json:"cpuUsedRatio"`
	Memory                int64             `json:"memory"`
	MemoryUsed            int64             `json:"memoryUsed"`
	MemoryUsedRatio       string            `json:"memoryUsedRatio"`
	Pod                   int64             `json:"pod"`
	PodUsed               int64             `json:"podUsed"`
	PodUsedRatio          string            `json:"podUsedRatio"`
}

func (n Node) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (n *Node) HasRole(role NodeRole) bool {
	for _, role_ := range n.Roles {
		if role == role_ {
			return true
		}
	}
	return false
}
