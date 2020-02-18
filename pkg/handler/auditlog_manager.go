package handler

import (
	"sort"

	"github.com/zdnscloud/singlecloud/pkg/auditlog/storage"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
)

type AuditLogManager struct {
	audit storage.StorageDriver
}

func newAuditLogManager(auditlogDriver storage.StorageDriver) *AuditLogManager {
	return &AuditLogManager{
		audit: auditlogDriver,
	}
}

func (a *AuditLogManager) List(ctx *resource.Context) interface{} {
	logs, err := a.audit.List()
	if err != nil {
		log.Warnf("list auditlog failed %s", err.Error())
		return nil
	}

	sort.Sort(logs)
	return logs
}
