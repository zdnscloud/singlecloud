package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetCronJobSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

type CronJob struct {
	resttypes.Resource `json:",inline"`
	Name               string        `json:"name,omitempty"`
	Schedule           string        `json:"schedule,omitempty"`
	Containers         []Container   `json:"containers"`
	Status             CronJobStatus `json:"status"`
}

type CronJobStatus struct {
	LastScheduleTime resttypes.ISOTime `json:"lastScheduleTime,omitempty"`
	Active           []ObjectReference `json:"objectReference,omitempty"`
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

var CronJobType = resttypes.GetResourceType(CronJob{})
