package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type HorizontalPodAutoscaler struct {
	resource.ResourceBase    `json:",inline"`
	Name                     string                        `json:"name"`
	ScaleTargetKind          string                        `json:"scaleTargetKind"`
	ScaleTargetName          string                        `json:"scaleTargetName"`
	MinReplicas              int                           `json:"minReplicas"`
	MaxReplicas              int                           `json:"maxReplicas"`
	CPUUtilizationPercentage int                           `json:"cpuUtilizationPercentage"`
	Status                   HorizontalPodAutoscalerStatus `json:"status,omitempty"`
}

func (H HorizontalPodAutoscaler) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

type HorizontalPodAutoscalerStatus struct {
	CurrentReplicas                 int `json:"currentReplicas,omitempty"`
	DesiredReplicas                 int `json:"desiredReplicas,omitempty"`
	CurrentCPUUtilizationPercentage int `json:"currentCPUUtilizationPercentage,omitempty"`
}
