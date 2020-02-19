package auditlog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zdnscloud/gorest"
	resterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/auditlog/storage"
	"github.com/zdnscloud/singlecloud/pkg/db"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AuditLogTable    = "auditlog"
	MaxAuditLogCount = 1000

	OperationTypeCreate = "create"
	OperationTypeUpdate = "update"
	OperationTypeDelete = "delete"
)

type AuditLogger struct {
	Storage storage.StorageDriver
}

func New() (*AuditLogger, error) {
	a := &AuditLogger{}

	tn, _ := kvzoo.TableNameFromSegments(AuditLogTable)
	table, err := db.GetGlobalDB().CreateOrGetTable(tn)
	if err != nil {
		return nil, fmt.Errorf("create or get db table %s failed %s", tn, err.Error())
	}

	driver, err := storage.NewDefaultDriver(table, MaxAuditLogCount)
	if err != nil {
		return a, err
	}

	a.Storage = driver
	return a, nil
}

func (a *AuditLogger) AuditHandler() gorest.HandlerFunc {
	return func(ctx *resource.Context) *resterr.APIError {
		log := &types.AuditLog{
			User:          getCurrentUser(ctx),
			SourceAddress: ctx.Request.Host,
			ResourceKind:  resource.DefaultKindName(ctx.Resource),
			ResourcePath:  ctx.Request.URL.Path,
		}

		switch ctx.Request.Method {
		case http.MethodPost:
			log.Operation = OperationTypeCreate
		case http.MethodPut:
			log.Operation = OperationTypeUpdate
		case http.MethodDelete:
			log.Operation = OperationTypeDelete
		default:
			return nil
		}

		if action := ctx.Resource.GetAction(); action != nil {
			log.Operation = action.Name
			detail, err := json.Marshal(action.Input)
			if err != nil {
				return resterr.NewAPIError(resterr.InvalidBodyContent, fmt.Sprintf("record audit log failed marshal action input err %s", err.Error()))
			}
			log.Detail = string(detail)
		} else {
			detail, err := json.Marshal(ctx.Resource)
			if err != nil {
				return resterr.NewAPIError(resterr.InvalidBodyContent, fmt.Sprintf("record audit log failed marshal resource err %s", err.Error()))
			}
			log.Detail = string(detail)
		}

		log.SetCreationTimestamp(time.Now())
		if err := a.Storage.Add(log); err != nil {
			return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("record audit log failed %s", err.Error()))
		}
		return nil
	}
}

func getCurrentUser(ctx *resource.Context) string {
	currentUser := ctx.Request.Context().Value(types.CurrentUserKey)
	if currentUser == nil {
		return ""
	}
	return currentUser.(string)
}
