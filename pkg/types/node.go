package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type NodeStatus string

const (
	NSReady    NodeStatus = "Ready"
	NSNotReady NodeStatus = "NotReady"
	NSCordoned NodeStatus = "Cordoned"
	NSDrained  NodeStatus = "Drained"
)

type NodeRole string

const (
	RoleControlPlane NodeRole = "controlplane"
	RoleEtcd         NodeRole = "etcd"
	RoleWorker       NodeRole = "worker"
	RoleEdge         NodeRole = "edge"
	RoleStorage      NodeRole = "storage"
)

const (
	NodeCordon   string = "cordon"
	NodeUnCordon string = "uncordon"
	NodeDrain    string = "drain"
)

type Node struct {
	resource.ResourceBase `json:",inline"`
	Name                  string            `json:"name" rest:"required=true,description=immutable,isDomain=true"`
	Status                NodeStatus        `json:"status" rest:"description=readonly"`
	Address               string            `json:"address,omitempty" rest:"required=true,description=immutable"`
	Roles                 []NodeRole        `json:"roles,omitempty" rest:"required=true,options=controlplane|etcd|worker|edge"`
	Labels                map[string]string `json:"labels,omitempty" rest:"description=readonly"`
	Annotations           map[string]string `json:"annotations,omitempty" rest:"description=readonly"`
	OperatingSystem       string            `json:"operatingSystem,omitempty" rest:"description=readonly"`
	OperatingSystemImage  string            `json:"operatingSystemImage,omitempty" rest:"description=readonly"`
	DockerVersion         string            `json:"dockerVersion,omitempty" rest:"description=readonly"`
	Cpu                   int64             `json:"cpu" rest:"description=readonly"`
	CpuUsed               int64             `json:"cpuUsed" rest:"description=readonly"`
	CpuUsedRatio          string            `json:"cpuUsedRatio" rest:"description=readonly"`
	Memory                int64             `json:"memory" rest:"description=readonly"`
	MemoryUsed            int64             `json:"memoryUsed" rest:"description=readonly"`
	MemoryUsedRatio       string            `json:"memoryUsedRatio" rest:"description=readonly"`
	Pod                   int64             `json:"pod" rest:"description=readonly"`
	PodUsed               int64             `json:"podUsed" rest:"description=readonly"`
	PodUsedRatio          string            `json:"podUsedRatio" rest:"description=readonly"`
}

func (n Node) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (n Node) GetActions() []resource.Action {
	return []resource.Action{
		resource.Action{
			Name: NodeCordon,
		},
		resource.Action{
			Name: NodeUnCordon,
		},
		resource.Action{
			Name: NodeDrain,
		},
	}
}

func (n *Node) HasRole(role NodeRole) bool {
	for _, role_ := range n.Roles {
		if role == role_ {
			return true
		}
	}
	return false
}
