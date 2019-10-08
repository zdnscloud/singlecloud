package gorest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"reflect"

	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
)

func restHandler(ctx *resource.Context) *goresterr.APIError {
	if ctx.Resource.GetAction() != nil {
		return handleAction(ctx)
	}

	switch ctx.Method {
	case http.MethodGet:
		return handleList(ctx)
	case http.MethodPost:
		return handleCreate(ctx)
	case http.MethodPut:
		return handleUpdate(ctx)
	case http.MethodDelete:
		return handleDelete(ctx)
	default:
		return goresterr.NewAPIError(goresterr.NotFound, "no found request handler")
	}
}

func handleCreate(ctx *resource.Context) *goresterr.APIError {
	schema := ctx.Resource.GetSchema()
	handler := schema.GetHandler().GetCreateHandler()
	if handler == nil {
		return goresterr.NewAPIError(goresterr.NotFound, "no handler for create")
	}

	r, err := handler(ctx)
	if err != nil {
		return err
	}

	ctx.Resource.SetID(r.GetID())
	r.SetType(ctx.Resource.GetType())
	httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
	if err := schema.AddLinksToResource(r, httpSchemeAndHost); err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
	}
	WriteResponse(ctx.Response, http.StatusCreated, r)
	return nil
}

func handleDelete(ctx *resource.Context) *goresterr.APIError {
	handler := ctx.Resource.GetSchema().GetHandler().GetDeleteHandler()
	if handler == nil {
		return goresterr.NewAPIError(goresterr.NotFound, "no handler for delete")
	}

	if err := handler(ctx); err != nil {
		return err
	}

	WriteResponse(ctx.Response, http.StatusNoContent, nil)
	return nil
}

func handleUpdate(ctx *resource.Context) *goresterr.APIError {
	schema := ctx.Resource.GetSchema()
	handler := schema.GetHandler().GetUpdateHandler()
	if handler == nil {
		return goresterr.NewAPIError(goresterr.NotFound, "no handler for update")
	}

	r, err := handler(ctx)
	if err != nil {
		return err
	}

	httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
	if err := schema.AddLinksToResource(r, httpSchemeAndHost); err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
	}
	r.SetType(ctx.Resource.GetType())
	WriteResponse(ctx.Response, http.StatusOK, r)
	return nil
}

func handleList(ctx *resource.Context) *goresterr.APIError {
	var result interface{}
	schema := ctx.Resource.GetSchema()
	if ctx.Resource.GetID() == "" {
		handler := schema.GetHandler().GetListHandler()
		if handler == nil {
			return goresterr.NewAPIError(goresterr.NotFound, "no found for list")
		}

		data := handler(ctx)
		rc, err := resource.NewResourceCollection(ctx.Resource, data)
		if err != nil {
			return goresterr.NewAPIError(goresterr.ServerError, err.Error())
		}

		httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
		if err := schema.AddLinksToResourceCollection(rc, httpSchemeAndHost); err != nil {
			return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
		}
		result = rc
	} else {
		handler := schema.GetHandler().GetGetHandler()
		if handler == nil {
			return goresterr.NewAPIError(goresterr.NotFound, "no found for list")
		}
		r := handler(ctx)
		if r == nil || (reflect.ValueOf(r).Kind() == reflect.Ptr && reflect.ValueOf(r).IsNil()) {
			return goresterr.NewAPIError(goresterr.NotFound,
				fmt.Sprintf("%s resource with id %s doesn't exist", ctx.Resource.GetType(), ctx.Resource.GetID()))
		} else {
			//the resource handler returns mayn't include schema
			r.SetSchema(ctx.Resource.GetSchema())
			r.SetParent(ctx.Resource.GetParent())
			httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
			if err := schema.AddLinksToResource(r, httpSchemeAndHost); err != nil {
				return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
			}
			r.SetType(ctx.Resource.GetType())
		}
		result = r
	}

	WriteResponse(ctx.Response, http.StatusOK, result)
	return nil
}

func handleAction(ctx *resource.Context) *goresterr.APIError {
	handler := ctx.Resource.GetSchema().GetHandler().GetActionHandler()
	if handler == nil {
		return goresterr.NewAPIError(goresterr.NotFound, "no handler for action")
	}

	result, err := handler(ctx)
	if err != nil {
		return err
	}

	WriteResponse(ctx.Response, http.StatusOK, result)
	return nil
}

const ContentTypeKey = "Content-Type"

func WriteResponse(resp http.ResponseWriter, status int, result interface{}) {
	var body []byte
	resp.Header().Set(ContentTypeKey, "application/json")
	body, _ = json.Marshal(result)
	resp.WriteHeader(status)
	resp.Write(body)
}
