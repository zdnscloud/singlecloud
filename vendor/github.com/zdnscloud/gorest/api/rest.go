package api

import (
	"net/http"

	"github.com/zdnscloud/gorest/api/handler"
	"github.com/zdnscloud/gorest/types"
)

func RestHandler(ctx *types.Context) *types.APIError {
	if ctx.Action != nil {
		return handler.ActionHandler(ctx)
	}

	var reqHandler types.RequestHandler
	switch ctx.Method {
	case http.MethodGet:
		reqHandler = handler.ListHandler
	case http.MethodPost:
		reqHandler = handler.CreateHandler
	case http.MethodPut:
		reqHandler = handler.UpdateHandler
	case http.MethodDelete:
		reqHandler = handler.DeleteHandler
	}

	if reqHandler == nil {
		return types.NewAPIError(types.NotFound, "no found request handler")
	} else {
		return reqHandler(ctx)
	}
}
