package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type ScaleTargetKind string

const (
	ScaleTargetKindDeployment  ScaleTargetKind = "deployment"
	ScaleTargetKindStatefulSet ScaleTargetKind = "statefulset"
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

type ResourceMetricSpec struct {
	ResourceName       ResourceName     `json:"resourceName,omitempty" rest:"options=cpu|memory"`
	TargetType         MetricTargetType `json:"targetType,omitempty" rest:"options=Utilization|AverageValue"`
	AverageValue       string           `json:"averageValue,omitempty"`
	AverageUtilization int              `json:"averageUtilization,omitempty"`
}

type CustomMetricSpec struct {
	MetricName   string            `json:"metricName,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	AverageValue string            `json:"averageValue,omitempty"`
}

type HorizontalPodAutoscalerStatus struct {
	CurrentReplicas int          `json:"currentReplicas,omitempty"`
	DesiredReplicas int          `json:"desiredReplicas,omitempty"`
	CurrentMetrics  MetricStatus `json:"currentMetrics,omitempty"`
}

type MetricStatus struct {
	ResourceMetrics []ResourceMetricSpec `json:"resourceMetrics,omitempty"`
	CustomMetrics   []CustomMetricSpec   `json:"customMetrics,omitempty"`
}

type HorizontalPodAutoscaler struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                        `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	ScaleTargetKind       ScaleTargetKind               `json:"scaleTargetKind" rest:"required=true,options=deployment|statefulset"`
	ScaleTargetName       string                        `json:"scaleTargetName" rest:"required=true,isDomain=true"`
	MinReplicas           int                           `json:"minReplicas"`
	MaxReplicas           int                           `json:"maxReplicas" rest:"required=true"`
	ResourceMetrics       []ResourceMetricSpec          `json:"resourceMetrics,omitempty"`
	CustomMetrics         []CustomMetricSpec            `json:"customMetrics,omitempty"`
	Status                HorizontalPodAutoscalerStatus `json:"status,omitempty" rest:"description=readonly"`
}

func (H HorizontalPodAutoscaler) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
