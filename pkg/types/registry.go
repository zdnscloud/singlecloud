package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Registry struct {
	resource.ResourceBase `json:",inline"`
	Cluster               string `json:"cluster" rest:"required=true,minLen=1,maxLen=128"`
	IngressDomain         string `json:"ingressDomain"`
	StorageClass          string `json:"storageClass"`
	StorageSize           int    `json:"storageSize"`
	AdminPassword         string `json:"adminPassword"`
	RedirectUrl           string `json:"redirectUrl"`
}
