package types

import (
	"time"

	resttypes "github.com/zdnscloud/gorest/types"
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

func SetUserQuotaSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "PUT", "DELETE", "POST"}
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionApproval,
		Input: ClusterInfo{},
	})
	schema.ResourceActions = append(schema.ResourceActions, resttypes.Action{
		Name:  ActionRejection,
		Input: Rejection{},
	})
}

type ClusterInfo struct {
	ClusterName string `json:"clusterName"`
}

type Rejection struct {
	Reason string `reason`
}

type UserQuota struct {
	resttypes.Resource `json:",inline"`
	Name               string            `json:"name,omitempty"`
	ClusterName        string            `json:"clusterName,omitempty"`
	Namespace          string            `json:"namespace"`
	UserName           string            `json:"userName"`
	CPU                string            `json:"cpu"`
	Memory             string            `json:"memory"`
	Storage            string            `json:"storage"`
	RequestType        string            `json:"requestType"`
	Status             string            `json:"status"`
	Purpose            string            `json:"purpose"`
	Requestor          string            `json:"requestor,omitempty"`
	Telephone          string            `json:"telephone,omitempty"`
	RejectionReason    string            `json:"rejectionReason,omitempty"`
	ResponseTimestamp  resttypes.ISOTime `json:"responseTimestamp,omitempty"`
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
		return u[i].UserName < u[j].UserName
	} else {
		return time.Time(u[i].CreationTimestamp).Before(time.Time(u[j].CreationTimestamp))
	}
}

var UserQuotaType = resttypes.GetResourceType(UserQuota{})
