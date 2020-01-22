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
	ClusterName string `json:"clusterName" rest:"required=true,isDomain=true"`
}

type Rejection struct {
	Reason string `json:"reason"`
}

type UserQuota struct {
	resource.ResourceBase `json:",inline"`
	Name                  string           `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	ClusterName           string           `json:"clusterName,omitempty" rest:"isDomain=true"`
	Namespace             string           `json:"namespace" rest:"required=true,isDomain=true"`
	UserName              string           `json:"userName" rest:"description=readonly"`
	CPU                   string           `json:"cpu"`
	Memory                string           `json:"memory"`
	Storage               string           `json:"storage"`
	RequestType           string           `json:"requestType" rest:"description=readonly"`
	Status                string           `json:"status" rest:"description=readonly"`
	Purpose               string           `json:"purpose"`
	Requestor             string           `json:"requestor,omitempty"`
	Telephone             string           `json:"telephone,omitempty"`
	RejectionReason       string           `json:"rejectionReason,omitempty"`
	ResponseTimestamp     resource.ISOTime `json:"responseTimestamp,omitempty" rest:"description=readonly"`
}

var UserQuotaActions = []resource.Action{
	resource.Action{
		Name:  ActionApproval,
		Input: &ClusterInfo{},
	},
	resource.Action{
		Name:  ActionRejection,
		Input: &Rejection{},
	},
}

func (uq UserQuota) GetActions() []resource.Action {
	return UserQuotaActions
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
