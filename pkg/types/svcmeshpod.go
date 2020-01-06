package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type SvcMeshPod struct {
	resource.ResourceBase `json:",inline"`
	Stat                  Stat  `json:"stat,omitempty"`
	Inbound               Stats `json:"inbound,omitempty"`
	Outbound              Stats `json:"outbound,omitempty"`
}

func (p SvcMeshPod) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{SvcMeshWorkload{}}
}

type SvcMeshPods []*SvcMeshPod

type Resource struct {
	Namespace string `json:"namespace,omitempty"`
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
}

type Stats []Stat

type Stat struct {
	ID              string           `json:"id,omitempty"`
	WorkloadID      string           `json:"workloadId,omitempty"`
	Link            string           `json:"link,omitempty"`
	Resource        Resource         `json:"resource,omitempty"`
	TimeWindow      string           `json:"timeWindow,omitempty"`
	Status          string           `json:"status,omitempty"`
	MeshedPodCount  int              `json:"meshedPodCount,omitempty"`
	RunningPodCount int              `json:"runningPodCount,omitempty"`
	FailedPodCount  int              `json:"failedPodCount,omitempty"`
	BasicStat       BasicStat        `json:"basicStat,omitempty"`
	TcpStat         TcpStat          `json:"tcpStat,omitempty"`
	TsStat          TrafficSplitStat `json:"trafficSplitStat,omitempty"`
	PodErrors       PodErrors        `json:"podErrors,omitempty"`
}

type BasicStat struct {
	SuccessCount       int `json:"successCount,omitempty"`
	FailureCount       int `json:"failureCount,omitempty"`
	LatencyMsP50       int `json:"latencyMsP50,omitempty"`
	LatencyMsP95       int `json:"latencyMsP95,omitempty"`
	LatencyMsP99       int `json:"latencyMsP99,omitempty"`
	ActualSuccessCount int `json:"actualSuccessCount,omitempty"`
	ActualFailureCount int `json:"actualFailureCount,omitempty"`
}

type TcpStat struct {
	OpenConnections int `json:"openConnections,omitempty"`
	ReadBytesTotal  int `json:"readBytesTotal,omitempty"`
	WriteBytesTotal int `json:"writeBytesTotal,omitempty"`
}

type TrafficSplitStat struct {
	Apex   string `json:"apex,omitempty"`
	Leaf   string `json:"leaf,omitempty"`
	Weight string `json:"weight,omitempty"`
}

type PodErrors []PodError

type PodError struct {
	PodName string           `json:"podName,omitempty"`
	Errors  []ContainerError `json:"errors,omitempty"`
}

type ContainerError struct {
	Message   string `json:"message,omitempty"`
	Container string `json:"container,omitempty"`
	Image     string `json:"image,omitempty"`
	Reason    string `json:"reason,omitempty"`
}
