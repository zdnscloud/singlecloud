package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetMonitorSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"DELETE"}
	schema.Parents = []string{ClusterType}
}

type Monitor struct {
	resttypes.Resource  `json:",inline"`
	IngressDomain       string `json:"ingressDomain"`
	StorageClass        string `json:"storageClass"`
	StorageSize         int    `json:"storageSize"`
	PrometheusRetention int    `json:"prometheusRetention"`
	ScrapeInterval      int    `json:"scrapeInterval"`
	AdminPassword       string `json:"adminPassword"`
	RedirectUrl         string `json:"redirectUrl"`
}

var MonitorType = resttypes.GetResourceType(Monitor{})
