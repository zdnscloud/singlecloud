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
	LastScheduleTime resource.ISOTime  `json:"lastScheduleTime,omitempty" rest:"description=readonly"`
	ObjectReferences []ObjectReference `json:"objectReferences,omitempty" rest:"description=readonly"`
}

type ObjectReference struct {
	Kind            string `json:"kind,omitempty" rest:"description=readonly"`
	Namespace       string `json:"namespace,omitempty" rest:"description=readonly"`
	Name            string `json:"name,omitempty" rest:"description=readonly"`
	UID             string `json:"uid,omitempty" rest:"description=readonly"`
	APIVersion      string `json:"apiVersion,omitempty" rest:"description=readonly"`
	ResourceVersion string `json:"resourceVersion,omitempty" rest:"description=readonly"`
	FieldPath       string `json:"fieldPath,omitempty" rest:"description=readonly"`
}

func (c CronJob) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
