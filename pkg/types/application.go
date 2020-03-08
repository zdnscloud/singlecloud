package types

import (
	"encoding/json"

	"github.com/zdnscloud/gorest/resource"
)

type Application struct {
	resource.ResourceBase `json:",inline"`
	Name                  string          `json:"name" rest:"required=true,isDomain=true"`
	ChartName             string          `json:"chartName" rest:"required=true"`
	ChartVersion          string          `json:"chartVersion" rest:"required=true"`
	ChartIcon             string          `json:"chartIcon" rest:"description=readonly"`
	Status                string          `json:"status" rest:"description=readonly"`
	WorkloadCount         int             `json:"workloadCount,omitempty" rest:"description=readonly"`
	ReadyWorkloadCount    int             `json:"readyWorkloadCount,omitempty" rest:"description=readonly"`
	AppResources          AppResources    `json:"appResources,omitempty" rest:"description=readonly"`
	Configs               json.RawMessage `json:"configs,omitempty"`
	InjectServiceMesh     bool            `json:"injectServiceMesh,omitempty"`
}

type AppResource struct {
	Namespace         string           `json:"namespace"`
	Name              string           `json:"name"`
	Type              string           `json:"type"`
	Link              string           `json:"link,omitempty"`
	Replicas          int              `json:"replicas,omitempty"`
	ReadyReplicas     int              `json:"readyReplicas,omitempty"`
	Exists            bool             `json:"exists,omitempty"`
	CreationTimestamp resource.ISOTime `json:"creationTimestamp,omitempty"`
}

func (a Application) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (a Application) SupportAsyncDelete() bool {
	return true
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
