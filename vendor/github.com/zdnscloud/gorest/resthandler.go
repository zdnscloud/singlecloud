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

	resource, err := handler(ctx)
	if err != nil {
		return err
	}

	ctx.Resource.SetID(resource.GetID())
	httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
	if links, err := schema.GenerateLinks(ctx.Resource, httpSchemeAndHost); err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
	} else {
		resource.SetLinks(links)
	}
	resource.SetType(ctx.Resource.GetType())
	writeResponse(ctx.Response, http.StatusCreated, resource)
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

	writeResponse(ctx.Response, http.StatusNoContent, nil)
	return nil
}

func handleUpdate(ctx *resource.Context) *goresterr.APIError {
	schema := ctx.Resource.GetSchema()
	handler := schema.GetHandler().GetUpdateHandler()
	if handler == nil {
		return goresterr.NewAPIError(goresterr.NotFound, "no handler for update")
	}

	resource, err := handler(ctx)
	if err != nil {
		return err
	}

	httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
	if links, err := schema.GenerateLinks(ctx.Resource, httpSchemeAndHost); err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
	} else {
		resource.SetLinks(links)
	}
	resource.SetType(ctx.Resource.GetType())
	writeResponse(ctx.Response, http.StatusOK, resource)
	return nil
}

type Collection struct {
	Type         string                                              `json:"type,omitempty"`
	ResourceType string                                              `json:"resourceType,omitempty"`
	Links        map[resource.ResourceLinkType]resource.ResourceLink `json:"links,omitempty"`
	Data         interface{}                                         `json:"data"`
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
		if data == nil {
			data = make([]resource.Resource, 0)
		} else {
			//check return slice is a slice of resource
			value := reflect.ValueOf(data)
			if value.Kind() != reflect.Slice {
				return goresterr.NewAPIError(goresterr.ServerError,
					fmt.Sprintf("list handler doesn't return slice but %v", reflect.ValueOf(data).Kind()))
			}
			if value.Len() > 0 {
				elem := value.Index(0)
				if _, ok := elem.Interface().(resource.Resource); ok == false {
					return goresterr.NewAPIError(goresterr.ServerError,
						fmt.Sprintf("list handler doesn't return slice of resource but %v", elem.Kind()))
				}
			}
		}

		httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
		if links, err := schema.GenerateLinks(ctx.Resource, httpSchemeAndHost); err != nil {
			return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
		} else {
			collection := &Collection{
				Type:         "collection",
				ResourceType: ctx.Resource.GetType(),
				Data:         data,
				Links:        links,
			}
			result = collection
		}
	} else {
		handler := schema.GetHandler().GetGetHandler()
		if handler == nil {
			return goresterr.NewAPIError(goresterr.NotFound, "no found for list")
		}
		result = handler(ctx)
		if result == nil {
			return goresterr.NewAPIError(goresterr.NotFound,
				fmt.Sprintf("%s resource with id %s doesn't exist", ctx.Resource.GetType(), ctx.Resource.GetID()))
		} else {
			resource, ok := result.(resource.Resource)
			if ok == false {
				return goresterr.NewAPIError(goresterr.ServerError,
					fmt.Sprintf("get handler doesn't return resource but %v", reflect.ValueOf(result).Kind()))
			}
			httpSchemeAndHost := path.Join(ctx.Request.URL.Scheme, ctx.Request.URL.Host)
			if links, err := schema.GenerateLinks(ctx.Resource, httpSchemeAndHost); err != nil {
				return goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("generate links failed:%s", err.Error()))
			} else {
				resource.SetLinks(links)
			}
		}
	}

	writeResponse(ctx.Response, http.StatusOK, result)
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

	writeResponse(ctx.Response, http.StatusOK, result)
	return nil
}

const ContentTypeKey = "Content-Type"

func writeResponse(resp http.ResponseWriter, status int, result interface{}) {
	var body []byte
	resp.Header().Set(ContentTypeKey, "application/json")
	body, _ = json.Marshal(result)
	resp.WriteHeader(status)
	resp.Write(body)
}
