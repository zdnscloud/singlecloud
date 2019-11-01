package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	DefaultMonitorStorageClass        = "lvm"
	DefaultMonitorStorageSize         = 10
	DefaultMonitorPrometheusRetention = 10
	DefaultMonitorScrapeInterval      = 10
	DefaultMonitorAdminPassword       = "zcloud"
)

type Monitor struct {
	resource.ResourceBase `json:",inline"`
	IngressDomain         string `json:"ingressDomain"`
	StorageClass          string `json:"storageClass" rest:"options=lvm|cephfs"`
	StorageSize           int    `json:"storageSize"`
	PrometheusRetention   int    `json:"prometheusRetention"`
	ScrapeInterval        int    `json:"scrapeInterval"`
	AdminPassword         string `json:"adminPassword"`
	RedirectUrl           string `json:"redirectUrl"`
	Status                string `json:"status"`
}

func (m Monitor) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (m Monitor) CreateDefaultResource() resource.Resource {
	return &Monitor{
		StorageClass:        DefaultMonitorStorageClass,
		StorageSize:         DefaultMonitorStorageSize,
		PrometheusRetention: DefaultMonitorPrometheusRetention,
		ScrapeInterval:      DefaultMonitorScrapeInterval,
		AdminPassword:       DefaultMonitorAdminPassword,
	}
}
