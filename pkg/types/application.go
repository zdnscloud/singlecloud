package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetApplicationSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"DELETE"}
	schema.Parents = []string{NamespaceType}
}

type Application struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
	Version            int    `json:"version"`
	ChartName          string `json:"chartName"`
	ChartVersion       string `json:"chartVersion"`
	Configs            string `json:"configs,omitempty"`
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

var ApplicationType = resttypes.GetResourceType(Application{})
