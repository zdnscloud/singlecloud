package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Iscsi struct {
	resource.ResourceBase `json:",inline"`
	Name                  string        `json:"name" rest:"required=true,isDomain=true"`
	Target                string        `json:"target" rest:"required=true"`
	Port                  string        `json:"port" rest:"required=true"`
	Iqn                   string        `json:"iqn" rest:"required=true"`
	Chap                  bool          `json:"chap"`
	Username              string        `json:"username"`
	Password              string        `json:"password"`
	Initiators            []string      `json:"initiators" rest:"required=true"`
	Phase                 string        `json:"phase" rest:"description=readonly"`
	Size                  string        `json:"size" rest:"description=readonly"`
	UsedSize              string        `json:"usedSize" rest:"description=readonly"`
	FreeSize              string        `json:"freeSize" rest:"description=readonly"`
	Nodes                 []StorageNode `json:"nodes" rest:"description=readonly"`
	PVs                   []PV          `json:"pvs" rest:"description=readonly"`
}

func (s Iscsi) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}

func (s Iscsi) SupportAsyncDelete() bool {
	return true
}

type Iscsis []Iscsi

func (s Iscsis) Len() int           { return len(s) }
func (s Iscsis) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Iscsis) Less(i, j int) bool { return s[i].Name < s[j].Name }
