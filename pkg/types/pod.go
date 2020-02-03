package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Pod struct {
	resource.ResourceBase `json:",inline"`
	Name                  string      `json:"name"`
	NodeName              string      `json:"nodeName"`
	State                 string      `json:"state"`
	Containers            []Container `json:"containers"`
	Status                PodStatus   `json:"status,omitempty"`
}

type PodStatus struct {
	Phase             string            `json:"phase,omitempty"`
	StartTime         resource.ISOTime  `json:"startTime,omitempty"`
	HostIP            string            `json:"hostIP,omitempty"`
	PodIP             string            `json:"podIP,omitempty"`
	PodConditions     []PodCondition    `json:"podConditions,omitempty"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty"`
}

type PodCondition struct {
	Type               string           `json:"type,omitempty"`
	Status             string           `json:"status,omitempty"`
	LastProbeTime      resource.ISOTime `json:"lastProbeTime,omitempty"`
	LastTransitionTime resource.ISOTime `json:"lastTransitionTime,omitempty"`
}

type ContainerStatus struct {
	Name         string          `json:"name,omitempty"`
	Ready        bool            `json:"ready,omitempty"`
	RestartCount int32           `json:"restartCount"`
	Image        string          `json:"image,omitempty"`
	ImageID      string          `json:"imageID,omitempty"`
	ContainerID  string          `json:"containerID,omitempty"`
	LastState    *ContainerState `json:"lastState,omitempty"`
	State        *ContainerState `json:"state,omitempty"`
}

type ContainerState struct {
	Type        string           `json:"type,omitempty"`
	ContainerID string           `json:"containerID,omitempty"`
	ExitCode    int32            `json:"exitCode,omitempty"`
	Reason      string           `json:"reason,omitempty"`
	Message     string           `json:"message,omitempty"`
	StartedAt   resource.ISOTime `json:"startedAt,omitempty"`
	FinishedAt  resource.ISOTime `json:"finishedAt,omitempty"`
}

const (
	WaitingState    = "Waiting"
	RunningState    = "Running"
	TerminatedState = "Terminated"
)

func (p Pod) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Deployment{}, DaemonSet{}, StatefulSet{}, Job{}, CronJob{}}
}
