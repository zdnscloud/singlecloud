package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type OperationType string

const (
	OperationTypeCreate OperationType = "create"
	OperationTypeUpdate OperationType = "update"
	OperationTypeDelete OperationType = "delete"
)

type AuditLog struct {
	resource.ResourceBase `json:",inline"`
	UID                   uint64        `json:"uid"`
	User                  string        `json:"user"`
	SourceAddress         string        `json:"sourceAddress"`
	Operation             OperationType `json:"operation" rest:"options=create|update|delete"`
	ResourceKind          string        `json:"resourceKind"`
	ResourcePath          string        `json:"resourcePath"`
	Detail                string        `json:"detail"`
}

type AuditLogs []*AuditLog

func (s AuditLogs) Len() int           { return len(s) }
func (s AuditLogs) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s AuditLogs) Less(i, j int) bool { return s[i].UID < s[j].UID }
