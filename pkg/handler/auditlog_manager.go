package handler

import (
	"sort"

	"github.com/zdnscloud/singlecloud/pkg/auditlog/storage"
	"github.com/zdnscloud/singlecloud/pkg/types"

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

	user := getCurrentUser(ctx)
	if isAdmin(user) {
		return logs
	}

	userLogs := []*types.AuditLog{}
	for _, log := range logs {
		if log.User == user {
			userLogs = append(userLogs, log)
		}
	}
	return userLogs
}
