package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type WorkFlowTaskStatus struct {
	CurrentStatus  string           `json:"currentStatus" rest:"description=readonly"`
	Message        string           `json:"message,omitempty" rest:"description=readonly"`
	StartTime      resource.ISOTime `json:"startedTime,omitempty" rest:"description=readonly"`
	CompletionTime resource.ISOTime `json:"completionTime,omitempty" rest:"description=readonly"`
}

const (
	WorkFlowTaskStatusSucceed = "succeed"
	WorkFlowTaskStatusFailed  = "failed"
	WorkFlowTaskStatusRunning = "running"
)

type WorkFlowTask struct {
	resource.ResourceBase `json:",inline"`
	ImageTag              string             `json:"imageTag" rest:"required=true"`
	Pods                  []WorkFlowTaskPod  `json:"pods" rest:"description=readonly"`
	Status                WorkFlowTaskStatus `json:"status" rest:"description=readonly"`
}

type WorkFlowTaskPod struct {
	Name       string             `json:"name"`
	Containers []string           `json:"containers"`
	Status     WorkFlowTaskStatus `json:"status"`
}

func (w WorkFlowTask) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{WorkFlow{}}
}

func (w WorkFlowTask) SupportAsyncDelete() bool {
	return true
}
