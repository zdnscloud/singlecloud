package types

import (
	"encoding/json"

	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	AppStatusCreate  = "create"
	AppStatusDelete  = "delete"
	AppStatusFailed  = "failed"
	AppStatusSucceed = "succeed"
)

func SetApplicationSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"DELETE"}
	schema.Parents = []string{NamespaceType}
}

type Application struct {
	resttypes.Resource `json:",inline"`
	Name               string          `json:"name"`
	ChartName          string          `json:"chartName"`
	ChartVersion       string          `json:"chartVersion"`
	Status             string          `json:"status"`
	AppResources       []AppResource   `json:"appResources,omitempty"`
	Configs            json.RawMessage `json:"configs,omitempty"`
	Manifests          []Manifest      `json:"manifests,omitempty"`
	SystemChart        bool            `json:"systemChart,omitempty"`
}

type AppResource struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Link string `json:"link"`
}

type Manifest struct {
	File      string `json:"file,omitempty"`
	Content   string `json:"content,omitempty"`
	Duplicate bool   `json:"duplicate,omitempty"`
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

var ApplicationType = resttypes.GetResourceType(Application{})
