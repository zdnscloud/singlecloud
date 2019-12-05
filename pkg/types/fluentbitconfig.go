package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type FluentBitConfig struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"-"`
	Namespace             string `json:"-"`
	OwnerType             string `json:"-"`
	OwnerName             string `json:"-"`
	RegExp                string `json:"regexp" rest:"required=true"`
	Time_Key              string `json:"timeKey,omitempty"`
	Time_Format           string `json:"timeFormat,omitempty"`
}

func (e FluentBitConfig) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Deployment{}, DaemonSet{}, StatefulSet{}}
}
