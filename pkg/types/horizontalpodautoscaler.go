package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type ScaleTargetKind string

const (
	ScaleTargetKindDeployment  ScaleTargetKind = "deployment"
	ScaleTargetKindStatefulSet ScaleTargetKind = "statefulset"
)

type MetricSourceType string

const (
	MetricSourceTypeResource = "Resource"
	MetricSourceTypePods     = "Pods"
)

type ResourceName string

const (
	ResourceNameCPU    ResourceName = "cpu"
	ResourceNameMemory ResourceName = "memory"
)

type MetricTargetType string

const (
	MetricTargetTypeUtilization  MetricTargetType = "Utilization"
	MetricTargetTypeAverageValue MetricTargetType = "AverageValue"
)

type MetricSpec struct {
	Type         MetricSourceType `json:"type,omitempty"`
	MetricName   string           `json:"metricName,omitempty"`
	ResourceName ResourceName     `json:"resourceName,omitempty"`
	TargetType   MetricTargetType `json:"targetType,omitempty"`
	MetricValue  `json:",inline"`
}

type MetricValue struct {
	AverageValue       string `json:"averageValue,omitempty"`
	AverageUtilization int    `json:"averageUtilization,omitempty"`
}

type HorizontalPodAutoscalerStatus struct {
	CurrentReplicas int            `json:"currentReplicas,omitempty"`
	DesiredReplicas int            `json:"desiredReplicas,omitempty"`
	CurrentMetrics  []MetricStatus `json:"currentMetrics,omitempty"`
}

type MetricStatus struct {
	Type         MetricSourceType `json:"type,omitempty"`
	MetricName   string           `json:"metricName,omitempty"`
	ResourceName ResourceName     `json:"resourceName,omitempty"`
	MetricValue  `json:",inline"`
}

type HorizontalPodAutoscaler struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                        `json:"name"`
	ScaleTargetKind       ScaleTargetKind               `json:"scaleTargetKind" rest:"required=true,options=deployment|statefulset"`
	ScaleTargetName       string                        `json:"scaleTargetName" rest:"required=true"`
	MinReplicas           int                           `json:"minReplicas"`
	MaxReplicas           int                           `json:"maxReplicas" rest:"required=true"`
	Metrics               []MetricSpec                  `json:"metrics,omitempty"`
	Status                HorizontalPodAutoscalerStatus `json:"status,omitempty"`
}

func (H HorizontalPodAutoscaler) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
