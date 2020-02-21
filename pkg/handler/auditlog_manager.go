package handler

import (
	"github.com/zdnscloud/singlecloud/pkg/auditlog"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
)

type AuditLogManager struct {
	audit *auditlog.AuditLogger
}

func newAuditLogManager(audit *auditlog.AuditLogger) *AuditLogManager {
	return &AuditLogManager{
		audit: audit,
	}
}

func (a *AuditLogManager) List(ctx *resource.Context) interface{} {
	logs, err := a.audit.List(getCurrentUser(ctx))
	if err != nil {
		log.Warnf("list audit log failed %s", err.Error())
	}
	return logs
}
