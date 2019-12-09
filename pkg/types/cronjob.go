package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type CronJob struct {
	resource.ResourceBase `json:",inline"`
	Name                  string        `json:"name,omitempty"`
	Schedule              string        `json:"schedule,omitempty"`
	RestartPolicy         string        `json:"restartPolicy,omitempty"`
	Containers            []Container   `json:"containers"`
	Status                CronJobStatus `json:"status" rest:"description=readonly"`
}

type CronJobStatus struct {
	LastScheduleTime resource.ISOTime  `json:"lastScheduleTime,omitempty"`
	ObjectReferences []ObjectReference `json:"objectReferences,omitempty"`
}

type ObjectReference struct {
	Kind            string `json:"kind,omitempty"`
	Namespace       string `json:"namespace,omitempty"`
	Name            string `json:"name,omitempty"`
	UID             string `json:"uid,omitempty"`
	APIVersion      string `json:"apiVersion,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
	FieldPath       string `json:"fieldPath,omitempty"`
}

func (c CronJob) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
