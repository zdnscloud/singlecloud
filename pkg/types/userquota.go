package types

import (
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

var UserQuotaType = resttypes.GetResourceType(UserQuota{})
