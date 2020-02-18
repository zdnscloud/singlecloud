package auditlog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
		user := getCurrentUser(ctx)
		if user == "" {
			return resterr.NewAPIError(resterr.Unauthorized, fmt.Sprintf("record audit log failed user is unknowned"))
		}

		var opt types.OperationType
		switch ctx.Request.Method {
		case http.MethodPost:
			opt = types.OperationTypeCreate
		case http.MethodPut:
			opt = types.OperationTypeUpdate
		case http.MethodDelete:
			opt = types.OperationTypeDelete
		default:
			return nil
		}

		detail, err := json.Marshal(ctx.Resource)
		if err != nil {
			return resterr.NewAPIError(resterr.InvalidBodyContent, fmt.Sprintf("record audit log failed marshal resource err %s", err.Error()))
		}

		log := &types.AuditLog{
			User:          user,
			SourceAddress: ctx.Request.Host,
			Operation:     opt,
			ResourceKind:  resource.DefaultKindName(ctx.Resource),
			ResourcePath:  genResourcePath(ctx),
			Detail:        string(detail),
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

func genResourcePath(ctx *resource.Context) string {
	ancestors := resource.GetAncestors(ctx.Resource)
	ids := []string{"/"}
	for _, ancestor := range ancestors {
		ids = append(ids, ancestor.GetID())
	}
	ids = append(ids, ctx.Resource.GetID())
	return strings.Join(ids, "/")
}
