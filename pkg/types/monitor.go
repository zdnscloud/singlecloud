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
	IngressDomain         string `json:"ingressDomain" rest:"description=immutable"`
	StorageClass          string `json:"storageClass" rest:"options=lvm|cephfs,description=immutable"`
	StorageSize           int    `json:"storageSize" rest:"description=immutable"`
	PrometheusRetention   int    `json:"prometheusRetention" rest:"description=immutable"`
	ScrapeInterval        int    `json:"scrapeInterval" rest:"description=immutable"`
	AdminPassword         string `json:"adminPassword" rest:"description=immutable"`
	RedirectUrl           string `json:"redirectUrl" rest:"description=readonly"`
	Status                string `json:"status" rest:"description=readonly"`
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
