package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Nfs struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name" rest:"required=true"`
	Server                string `json:"server" rest:"required=true"`
	Path                  string `json:"path" rest:"required=true"`
	Phase                 string `json:"phase" rest:"description=readonly"`
	Size                  string `json:"size" rest:"description=readonly"`
	UsedSize              string `json:"usedSize" rest:"description=readonly"`
	FreeSize              string `json:"freeSize" rest:"description=readonly"`
	PVs                   []PV   `json:"pvs" rest:"description=readonly"`
}

func (s Nfs) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (s Nfs) SupportAsyncDelete() bool {
	return true
}

type Nfss []Nfs

func (s Nfss) Len() int           { return len(s) }
func (s Nfss) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Nfss) Less(i, j int) bool { return s[i].Name < s[j].Name }
