package types

import (
	"time"

	"github.com/zdnscloud/gorest/resource"
)

const (
	TypeUserQuotaCreate = "create"
	TypeUserQuotaUpdate = "update"

	StatusUserQuotaProcessing = "processing"
	StatusUserQuotaApproval   = "approval"
	StatusUserQuotaRejection  = "rejection"

	ActionApproval  = "approval"
	ActionRejection = "reject"
)

type ClusterInfo struct {
	ClusterName string `json:"clusterName"`
}

type Rejection struct {
	Reason string `reason`
}

type UserQuota struct {
	resource.ResourceBase `json:",inline"`
	Name                  string           `json:"name,omitempty"`
	ClusterName           string           `json:"clusterName,omitempty"`
	Namespace             string           `json:"namespace"`
	UserName              string           `json:"userName"`
	CPU                   string           `json:"cpu"`
	Memory                string           `json:"memory"`
	Storage               string           `json:"storage"`
	RequestType           string           `json:"requestType"`
	Status                string           `json:"status"`
	Purpose               string           `json:"purpose"`
	Requestor             string           `json:"requestor,omitempty"`
	Telephone             string           `json:"telephone,omitempty"`
	RejectionReason       string           `json:"rejectionReason,omitempty"`
	ResponseTimestamp     resource.ISOTime `json:"responseTimestamp,omitempty"`
}

func (uq UserQuota) CreateAction(name string) *resource.Action {
	switch name {
	case ActionApproval:
		return &resource.Action{
			Name:  ActionApproval,
			Input: &ClusterInfo{},
		}
	case ActionRejection:
		return &resource.Action{
			Name:  ActionRejection,
			Input: &Rejection{},
		}
	default:
		return nil
	}
}

type UserQuotas []*UserQuota

func (u UserQuotas) Len() int {
	return len(u)
}
func (u UserQuotas) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}
func (u UserQuotas) Less(i, j int) bool {
	if time.Time(u[i].CreationTimestamp).Equal(time.Time(u[j].CreationTimestamp)) {
		if u[i].UserName == u[j].UserName {
			return u[i].Name < u[j].Name
		} else {
			return u[i].UserName < u[j].UserName
		}
	} else {
		return time.Time(u[i].CreationTimestamp).Before(time.Time(u[j].CreationTimestamp))
	}
}
