package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/zdnscloud/gorest/types"
)

func CreateHandler(ctx *types.Context) *types.APIError {
	handler := ctx.Object.GetSchema().Handler
	if handler == nil {
		return types.NewAPIError(types.NotFound, "no handler for create")
	}

	content, err := parseCreateBody(ctx)
	if err != nil {
		return err
	}

	result, err := handler.Create(ctx, content)
	if err != nil {
		return err
	}

	addResourceLinks(ctx, result)
	WriteResponse(ctx, http.StatusCreated, result)
	return nil
}

func DeleteHandler(ctx *types.Context) *types.APIError {
	handler := ctx.Object.GetSchema().Handler
	if handler == nil {
		return types.NewAPIError(types.NotFound, "no handler for delete")
	}

	setRuntimeObject(ctx, createRuntimeObject(ctx))
	if err := handler.Delete(ctx); err != nil {
		return err
	}

	WriteResponse(ctx, http.StatusNoContent, nil)
	return nil
}

func UpdateHandler(ctx *types.Context) *types.APIError {
	handler := ctx.Object.GetSchema().Handler
	if handler == nil {
		return types.NewAPIError(types.NotFound, "no handler for update")
	}

	val := createRuntimeObject(ctx)
	if err := decodeBody(ctx.Request, val); err != nil {
		return err
	}

	setRuntimeObject(ctx, val)
	result, err := handler.Update(ctx)
	if err != nil {
		return err
	}

	addResourceLinks(ctx, result)
	WriteResponse(ctx, http.StatusOK, result)
	return nil
}

func ListHandler(ctx *types.Context) *types.APIError {
	handler := ctx.Object.GetSchema().Handler
	if handler == nil {
		return types.NewAPIError(types.NotFound, "no found for list")
	}

	setRuntimeObject(ctx, createRuntimeObject(ctx))

	var result interface{}
	if ctx.Object.GetID() == "" {
		data := handler.List(ctx)
		if data == nil || reflect.ValueOf(data).IsNil() {
			data = make([]types.Object, 0)
		}

		collection := &types.Collection{
			Type:         "collection",
			ResourceType: ctx.Object.GetType(),
			Data:         data,
		}
		addCollectionLinks(ctx, collection)
		result = collection
	} else {
		result = handler.Get(ctx)
		if result == nil || reflect.ValueOf(result).IsNil() {
			return types.NewAPIError(types.NotFound,
				fmt.Sprintf("%s resource with id %s doesn't exist", ctx.Object.GetType(), ctx.Object.GetID()))
		}
		addResourceLinks(ctx, result)
	}

	WriteResponse(ctx, http.StatusOK, result)
	return nil
}

func ActionHandler(ctx *types.Context) *types.APIError {
	handler := ctx.Object.GetSchema().Handler
	if handler == nil {
		return types.NewAPIError(types.NotFound, "no handler for action")
	}

	if ctx.Action.Input != nil {
		val := createRuntimeActionInput(ctx)
		if err := decodeBody(ctx.Request, val); err != nil {
			return err
		}

		setRuntimeActionInput(ctx, val)
	}
	setRuntimeObject(ctx, createRuntimeObject(ctx))
	result, err := handler.Action(ctx)
	if err != nil {
		return err
	}

	WriteResponse(ctx, http.StatusOK, result)
	return nil
}

func createRuntimeActionInput(ctx *types.Context) interface{} {
	val := reflect.ValueOf(ctx.Action.Input)
	valPtr := reflect.New(val.Type())
	valPtr.Elem().Set(val)
	return valPtr.Interface()
}

func setRuntimeActionInput(ctx *types.Context, val interface{}) {
	ctx.Action.Input = val
}

func createRuntimeObject(ctx *types.Context) interface{} {
	val := ctx.Object.GetSchema().StructVal
	valPtr := reflect.New(val.Type())
	valPtr.Elem().Set(val)
	return valPtr.Interface()
}

func decodeBody(req *http.Request, params interface{}) *types.APIError {
	reqBody, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		return types.NewAPIError(types.InvalidBodyContent,
			fmt.Sprintf("Failed to read request body: %v", err.Error()))
	}

	err = json.Unmarshal(reqBody, params)
	if err != nil {
		return types.NewAPIError(types.InvalidBodyContent,
			fmt.Sprintf("Failed to parse request body: %v", err.Error()))
	}

	return nil
}

func setRuntimeObject(ctx *types.Context, val interface{}) {
	objFromUrl := ctx.Object
	obj := val.(types.Object)
	obj.SetType(objFromUrl.GetType())
	obj.SetParent(objFromUrl.GetParent())
	obj.SetSchema(objFromUrl.GetSchema())
	obj.SetID(objFromUrl.GetID())
	ctx.Object = obj
}

func parseCreateBody(ctx *types.Context) ([]byte, *types.APIError) {
	var params struct {
		Yaml string `json:"yaml_"`
	}

	reqBody, err := ioutil.ReadAll(ctx.Request.Body)
	defer ctx.Request.Body.Close()
	if err != nil {
		return nil, types.NewAPIError(types.InvalidBodyContent,
			fmt.Sprintf("Failed to read request body: %v", err.Error()))
	}

	if err := json.Unmarshal(reqBody, &params); err != nil {
		return nil, types.NewAPIError(types.InvalidBodyContent,
			fmt.Sprintf("Failed to parse request body: %v", err.Error()))
	}

	val := createRuntimeObject(ctx)
	if err := json.Unmarshal(reqBody, val); err != nil {
		return nil, types.NewAPIError(types.InvalidBodyContent,
			fmt.Sprintf("Failed to parse request body: %v", err.Error()))
	}

	setRuntimeObject(ctx, val)
	return []byte(params.Yaml), CheckObjectFields(ctx)
}
