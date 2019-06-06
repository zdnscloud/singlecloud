package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetPodSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{DeploymentType, DaemonSetType, StatefulSetType, JobType, CronJobType}
}

type Pod struct {
	resttypes.Resource `json:",inline"`
	Name               string      `json:"name,omitempty"`
	NodeName           string      `json:"nodeName,omitempty"`
	State              string      `json:"state"`
	Containers         []Container `json:"containers"`
	Status             PodStatus   `json:"status"`
}

type PodStatus struct {
	Phase             string            `json:"phase,omitempty"`
	StartTime         resttypes.ISOTime `json:"startTime,omitempty"`
	HostIP            string            `json:"hostIP,omitempty"`
	PodIP             string            `json:"podIP,omitempty"`
	PodConditions     []PodCondition    `json:"podConditions,omitempty"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty"`
}

type PodCondition struct {
	Type               string            `json:"type,omitempty"`
	Status             string            `json:"status,omitempty"`
	LastProbeTime      resttypes.ISOTime `json:"lastProbeTime,omitempty"`
	LastTransitionTime resttypes.ISOTime `json:"lastTransitionTime,omitempty"`
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
	Type        string            `json:"type,omitempty"`
	ContainerID string            `json:"containerID,omitempty"`
	ExitCode    int32             `json:"exitCode,omitempty"`
	Reason      string            `json:"reason,omitempty"`
	Message     string            `json:"message,omitempty"`
	StartedAt   resttypes.ISOTime `json:"startedAt,omitempty"`
	FinishedAt  resttypes.ISOTime `json:"finishedAt,omitempty"`
}

const (
	WaitingState    = "Waiting"
	RunningState    = "Running"
	TerminatedState = "Terminated"
)

var PodType = resttypes.GetResourceType(Pod{})
