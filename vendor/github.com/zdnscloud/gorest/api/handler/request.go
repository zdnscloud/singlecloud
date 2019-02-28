package handler

import (
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/zdnscloud/gorest/httperror"
	"github.com/zdnscloud/gorest/parse"
	"github.com/zdnscloud/gorest/types"
)

func CreateHandler(apiContext *types.APIContext, next types.RequestHandler) error {
	handler := apiContext.Schema.Handler
	if handler == nil {
		return httperror.NewAPIError(httperror.NotFound, "no handler found")
	}

	object, err := parseRequestBody(apiContext)
	if err != nil {
		return err
	}

	result, err := handler.Create(object)
	if err != nil {
		return err
	}

	apiContext.WriteResponse(http.StatusCreated, result)
	return nil
}

func DeleteHandler(apiContext *types.APIContext, next types.RequestHandler) error {
	handler := apiContext.Schema.Handler
	if handler == nil {
		return httperror.NewAPIError(httperror.NotFound, "no handler found")
	}

	obj, err := getSchemaObject(apiContext)
	if err != nil {
		return nil
	}

	if apiContext.ID != "" {
		obj.SetID(apiContext.ID)
		err = handler.Delete(obj)
	} else {
		err = handler.BatchDelete(obj)
	}

	if err != nil {
		return err
	}
	apiContext.WriteResponse(http.StatusOK, nil)
	return nil
}

func UpdateHandler(apiContext *types.APIContext, next types.RequestHandler) error {
	handler := apiContext.Schema.Handler
	if handler == nil {
		return httperror.NewAPIError(httperror.NotFound, "no handler found")
	}

	object, err := parseRequestBody(apiContext)
	if err != nil {
		return err
	}

	oldObj, err := getSchemaObject(apiContext)
	if err != nil {
		return nil
	}

	oldObj.SetID(apiContext.ID)
	result, err := handler.Update(oldObj, oldObj, object)
	if err != nil {
		return err
	}

	apiContext.WriteResponse(http.StatusOK, result)
	return nil
}

func ListHandler(apiContext *types.APIContext, next types.RequestHandler) error {
	handler := apiContext.Schema.Handler
	if handler == nil {
		return httperror.NewAPIError(httperror.NotFound, "no handler found")
	}

	var result interface{}
	obj, err := getSchemaObject(apiContext)
	if err != nil {
		return nil
	}

	if apiContext.ID == "" {
		result = handler.List(obj)
	} else {
		obj.SetID(apiContext.ID)
		result = handler.Get(obj)
	}

	apiContext.WriteResponse(http.StatusOK, result)
	return nil
}

func ActionHandler(actionName string, action *types.Action, apiContext *types.APIContext) error {
	handler := apiContext.Schema.Handler
	if handler == nil {
		return httperror.NewAPIError(httperror.NotFound, "no handler found")
	}

	params, err := parseActionBody(apiContext)
	if err != nil {
		return err
	}

	obj, err := getSchemaObject(apiContext)
	if err != nil {
		return nil
	}

	obj.SetID(apiContext.ID)
	result, err := handler.Action(obj, apiContext.Action, params)
	if err != nil {
		return err
	}

	apiContext.WriteResponse(http.StatusOK, result)
	return nil
}

func getSchemaObject(apiContext *types.APIContext) (types.Object, error) {
	obj, ok := getSchemaStructVal(apiContext).(types.Object)
	if ok == false {
		return nil, httperror.NewAPIError(httperror.NotFound, "no found resource schema")
	}

	obj.SetType(apiContext.Schema.ID)
	obj.SetParent(apiContext.Schema.Parent)
	return obj, nil
}

func getSchemaStructVal(apiContext *types.APIContext) interface{} {
	val := apiContext.Schema.StructVal
	valPtr := reflect.New(val.Type())
	valPtr.Elem().Set(val)
	return valPtr.Interface()
}

func parseRequestBody(apiContext *types.APIContext) (types.Object, error) {
	decode := parse.GetDecoder(apiContext.Request, io.LimitReader(apiContext.Request.Body, parse.MaxFormSize))
	val := getSchemaStructVal(apiContext)
	if err := decode(val); err != nil {
		return nil, httperror.NewAPIError(httperror.InvalidBodyContent,
			fmt.Sprintf("Failed to parse body: %v", err))
	}

	if obj, ok := val.(types.Object); ok {
		obj.SetType(apiContext.Schema.ID)
		obj.SetParent(apiContext.Schema.Parent)
		return obj, nil
	} else {
		return nil, httperror.NewAPIError(httperror.InvalidBodyContent, fmt.Sprintf("Request Body mismatch resource schema"))
	}
}

func parseActionBody(apiContext *types.APIContext) (map[string]interface{}, error) {
	var params map[string]interface{}
	decode := parse.GetDecoder(apiContext.Request, io.LimitReader(apiContext.Request.Body, parse.MaxFormSize))
	if err := decode(&params); err != nil {
		return nil, httperror.NewAPIError(httperror.InvalidBodyContent,
			fmt.Sprintf("Failed to parse action body: %v", err))
	}

	return params, nil
}
