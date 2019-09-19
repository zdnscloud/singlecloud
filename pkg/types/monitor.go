package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Monitor struct {
	resource.ResourceBase `json:",inline"`
	IngressDomain         string `json:"ingressDomain"`
	StorageClass          string `json:"storageClass"`
	StorageSize           int    `json:"storageSize"`
	PrometheusRetention   int    `json:"prometheusRetention"`
	ScrapeInterval        int    `json:"scrapeInterval"`
	AdminPassword         string `json:"adminPassword"`
	RedirectUrl           string `json:"redirectUrl"`
}

func (m Monitor) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
