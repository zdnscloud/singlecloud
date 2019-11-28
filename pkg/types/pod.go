package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Pod struct {
	resource.ResourceBase `json:",inline"`
	Name                  string      `json:"name,omitempty" rest:"description=readonly"`
	NodeName              string      `json:"nodeName,omitempty" rest:"description=readonly"`
	State                 string      `json:"state" rest:"description=readonly"`
	Containers            []Container `json:"containers" rest:"description=readonly"`
	Status                PodStatus   `json:"status" rest:"description=readonly"`
}

type PodStatus struct {
	Phase             string            `json:"phase,omitempty" rest:"description=readonly"`
	StartTime         resource.ISOTime  `json:"startTime,omitempty" rest:"description=readonly"`
	HostIP            string            `json:"hostIP,omitempty" rest:"description=readonly"`
	PodIP             string            `json:"podIP,omitempty" rest:"description=readonly"`
	PodConditions     []PodCondition    `json:"podConditions,omitempty" rest:"description=readonly"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty" rest:"description=readonly"`
}

type PodCondition struct {
	Type               string           `json:"type,omitempty" rest:"description=readonly"`
	Status             string           `json:"status,omitempty" rest:"description=readonly"`
	LastProbeTime      resource.ISOTime `json:"lastProbeTime,omitempty" rest:"description=readonly"`
	LastTransitionTime resource.ISOTime `json:"lastTransitionTime,omitempty" rest:"description=readonly"`
}

type ContainerStatus struct {
	Name         string          `json:"name,omitempty" rest:"description=readonly"`
	Ready        bool            `json:"ready,omitempty" rest:"description=readonly"`
	RestartCount int32           `json:"restartCount" rest:"description=readonly"`
	Image        string          `json:"image,omitempty" rest:"description=readonly"`
	ImageID      string          `json:"imageID,omitempty" rest:"description=readonly"`
	ContainerID  string          `json:"containerID,omitempty" rest:"description=readonly"`
	LastState    *ContainerState `json:"lastState,omitempty" rest:"description=readonly"`
	State        *ContainerState `json:"state,omitempty" rest:"description=readonly"`
}

type ContainerState struct {
	Type        string           `json:"type,omitempty" rest:"description=readonly"`
	ContainerID string           `json:"containerID,omitempty" rest:"description=readonly"`
	ExitCode    int32            `json:"exitCode,omitempty" rest:"description=readonly"`
	Reason      string           `json:"reason,omitempty" rest:"description=readonly"`
	Message     string           `json:"message,omitempty" rest:"description=readonly"`
	StartedAt   resource.ISOTime `json:"startedAt,omitempty" rest:"description=readonly"`
	FinishedAt  resource.ISOTime `json:"finishedAt,omitempty" rest:"description=readonly"`
}

const (
	WaitingState    = "Waiting"
	RunningState    = "Running"
	TerminatedState = "Terminated"
)

func (p Pod) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Deployment{}, DaemonSet{}, StatefulSet{}, Job{}, CronJob{}}
}
