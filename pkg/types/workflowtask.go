package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type WorkFlowTaskStatus string

const (
	WorkFlowTaskStatusSucceed WorkFlowTaskStatus = "succeed"
	WorkFlowTaskStatusFailed  WorkFlowTaskStatus = "failed"
	WorkFlowTaskStatusRunning WorkFlowTaskStatus = "running"
)

type WorkFlowTask struct {
	resource.ResourceBase `json:",inline"`
	ImageTag              string             `json:"imageTag" rest:"required=true"`
	Status                WorkFlowTaskStatus `json:"status" rest:"description=readonly,options=succeed|failed|running"`
}

func (w WorkFlowTask) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{WorkFlow{}}
}
