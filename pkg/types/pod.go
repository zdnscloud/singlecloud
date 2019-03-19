package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetPodSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parent = DeploymentType
}

type Pod struct {
	resttypes.Resource `json:",inline"`
	Name               string             `json:"name,omitempty"`
	NodeName           string             `json:"nodeName,omitempty"`
	Containers         []Container        `json:"containers"`
	AdvancedOptions    PodAdvancedOptions `json:"advancedOptions"`
}

type PodAdvancedOptions struct {
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
	WaitingState    = "waiting"
	RunningState    = "running"
	TerminatedState = "terminated"
)

var PodType = resttypes.GetResourceType(Pod{})
