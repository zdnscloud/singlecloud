package types

import (
	"encoding/json"

	"github.com/zdnscloud/gorest/resource"
)

const (
	AppStatusCreate  = "create"
	AppStatusDelete  = "delete"
	AppStatusFailed  = "failed"
	AppStatusSucceed = "succeed"
)

var (
	ResourceTypeDeployment  = resource.DefaultKindName(Deployment{})
	ResourceTypeDaemonSet   = resource.DefaultKindName(DaemonSet{})
	ResourceTypeStatefulSet = resource.DefaultKindName(StatefulSet{})
	ResourceTypeJob         = resource.DefaultKindName(Job{})
	ResourceTypeCronJob     = resource.DefaultKindName(CronJob{})
	ResourceTypeConfigMap   = resource.DefaultKindName(ConfigMap{})
	ResourceTypeSecret      = resource.DefaultKindName(Secret{})
	ResourceTypeService     = resource.DefaultKindName(Service{})
	ResourceTypeIngress     = resource.DefaultKindName(Ingress{})
)

type Application struct {
	resource.ResourceBase `json:",inline"`
	Name                  string          `json:"name"`
	ChartName             string          `json:"chartName"`
	ChartVersion          string          `json:"chartVersion"`
	ChartIcon             string          `json:"chartIcon" rest:"description=readonly"`
	Status                string          `json:"status" rest:"description=readonly"`
	WorkloadCount         int             `json:"workloadCount,omitempty" rest:"description=readonly"`
	ReadyWorkloadCount    int             `json:"readyWorkloadCount,omitempty" rest:"description=readonly"`
	AppResources          AppResources    `json:"appResources,omitempty" rest:"description=readonly"`
	Configs               json.RawMessage `json:"configs,omitempty"`
	Manifests             []Manifest      `json:"manifests,omitempty" rest:"description=readonly"`
	SystemChart           bool            `json:"systemChart,omitempty" rest:"description=readonly"`
}

type AppResource struct {
	Name          string `json:"name" rest:"description=readonly"`
	Type          string `json:"type" rest:"description=readonly"`
	Link          string `json:"link" rest:"description=readonly"`
	Replicas      int    `json:"replicas,omitempty" rest:"description=readonly"`
	ReadyReplicas int    `json:"readyReplicas,omitempty" rest:"description=readonly"`
}

type Manifest struct {
	File      string `json:"file,omitempty" rest:"description=readonly"`
	Content   string `json:"content,omitempty" rest:"description=readonly"`
	Duplicate bool   `json:"duplicate,omitempty" rest:"description=readonly"`
}

func (a Application) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

type Applications []*Application

func (a Applications) Len() int {
	return len(a)
}

func (a Applications) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a Applications) Less(i, j int) bool {
	if a[i].ChartName == a[j].ChartName {
		if a[i].ChartVersion == a[j].ChartVersion {
			return a[i].Name < a[j].Name
		} else {
			return a[i].ChartVersion < a[j].ChartVersion
		}
	} else {
		return a[i].ChartName < a[j].ChartName
	}
}

type AppResources []AppResource

func (r AppResources) Len() int {
	return len(r)
}

func (r AppResources) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r AppResources) Less(i, j int) bool {
	if r[i].Type == r[j].Type {
		return r[i].Name < r[j].Name
	} else {
		return r[i].Type < r[j].Type
	}
}
