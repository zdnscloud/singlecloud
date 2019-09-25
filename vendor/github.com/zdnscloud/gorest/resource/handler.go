package resource

import (
	"fmt"
	"net/http"
	"reflect"

	goresterr "github.com/zdnscloud/gorest/error"
)

const (
	CreateMethod string = "Create"
	DeleteMethod string = "Delete"
	UpdateMethod string = "Update"
	ListMethod   string = "List"
	GetMethod    string = "Get"
	ActionMethod string = "Action"
)

type CreateHandler func(*Context) (Resource, *goresterr.APIError)
type DeleteHandler func(*Context) *goresterr.APIError
type UpdateHandler func(*Context) (Resource, *goresterr.APIError)
type ListHandler func(*Context) interface{}
type GetHandler func(*Context) Resource
type ActionHandler func(*Context) (interface{}, *goresterr.APIError)

type Handler interface {
	GetCreateHandler() CreateHandler
	GetDeleteHandler() DeleteHandler
	GetUpdateHandler() UpdateHandler
	GetListHandler() ListHandler
	GetGetHandler() GetHandler
	GetActionHandler() ActionHandler
}

func HandlerAdaptor(obj interface{}) (Handler, error) {
	handler := &DefaultHandler{}
	val := reflect.ValueOf(obj)
	hasAnyHandler := false
	if mv := val.MethodByName(ListMethod); mv.IsValid() {
		if method, ok := mv.Interface().(func(*Context) interface{}); ok {
			handler.listHandler = method
			hasAnyHandler = true
		}
	}

	if mv := val.MethodByName(GetMethod); mv.IsValid() {
		if method, ok := mv.Interface().(func(*Context) Resource); ok {
			handler.getHandler = method
			hasAnyHandler = true
		}
	}

	if mv := val.MethodByName(DeleteMethod); mv.IsValid() {
		if method, ok := mv.Interface().(func(*Context) *goresterr.APIError); ok {
			handler.deleteHandler = method
			hasAnyHandler = true
		}
	}

	if mv := val.MethodByName(UpdateMethod); mv.IsValid() {
		if method, ok := mv.Interface().(func(*Context) (Resource, *goresterr.APIError)); ok {
			handler.updateHandler = method
			hasAnyHandler = true
		}
	}

	if mv := val.MethodByName(CreateMethod); mv.IsValid() {
		if method, ok := mv.Interface().(func(*Context) (Resource, *goresterr.APIError)); ok {
			handler.createHandler = method
			hasAnyHandler = true
		}
	}

	if mv := val.MethodByName(ActionMethod); mv.IsValid() {
		if method, ok := mv.Interface().(func(*Context) (interface{}, *goresterr.APIError)); ok {
			handler.actionHandler = method
			hasAnyHandler = true
		}
	}

	if hasAnyHandler == false {
		return nil, fmt.Errorf("handler doesn't have any handle method")
	} else {
		return handler, nil
	}
}

var _ Handler = &DefaultHandler{}

type DefaultHandler struct {
	createHandler CreateHandler
	deleteHandler DeleteHandler
	updateHandler UpdateHandler
	listHandler   ListHandler
	getHandler    GetHandler
	actionHandler ActionHandler
}

func (h *DefaultHandler) GetCreateHandler() CreateHandler {
	return h.createHandler
}

func (h *DefaultHandler) GetDeleteHandler() DeleteHandler {
	return h.deleteHandler
}

func (h *DefaultHandler) GetUpdateHandler() UpdateHandler {
	return h.updateHandler
}

func (h *DefaultHandler) GetListHandler() ListHandler {
	return h.listHandler
}

func (h *DefaultHandler) GetGetHandler() GetHandler {
	return h.getHandler
}

func (h *DefaultHandler) GetActionHandler() ActionHandler {
	return h.actionHandler
}

func GetCollectionMethods(handler Handler) []HttpMethod {
	var collectionMethods []HttpMethod
	if handler.GetListHandler() != nil {
		collectionMethods = append(collectionMethods, http.MethodGet)
	}
	if handler.GetCreateHandler() != nil {
		collectionMethods = append(collectionMethods, http.MethodPost)
	}
	return collectionMethods
}

func GetResourceMethods(handler Handler) []HttpMethod {
	var resourceMethods []HttpMethod
	if handler.GetGetHandler() != nil {
		resourceMethods = append(resourceMethods, http.MethodGet)
	}
	if handler.GetDeleteHandler() != nil {
		resourceMethods = append(resourceMethods, http.MethodDelete)
	}
	if handler.GetUpdateHandler() != nil {
		resourceMethods = append(resourceMethods, http.MethodPut)
	}
	if handler.GetActionHandler() != nil {
		resourceMethods = append(resourceMethods, http.MethodPost)
	}
	return resourceMethods
}
