package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type CronJob struct {
	resource.ResourceBase `json:",inline"`
	Name                  string        `json:"name" rest:"required=true,isDomain=true"`
	Schedule              string        `json:"schedule" rest:"required=true"`
	RestartPolicy         string        `json:"restartPolicy" rest:"required=true,options=OnFailure|Never"`
	Containers            []Container   `json:"containers" rest:"required=true"`
	Status                CronJobStatus `json:"status,omitempty" rest:"description=readonly"`
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

func (c CronJob) SupportAsyncDelete() bool {
	return true
}
