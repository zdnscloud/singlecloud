package handler

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

type DefaultHandler struct {
}

func (m DefaultHandler) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	return nil, nil
}

func (m DefaultHandler) List(ctx *resttypes.Context) interface{} {
	return nil
}

func (m DefaultHandler) Get(ctx *resttypes.Context) interface{} {
	return nil
}

func (m DefaultHandler) Delete(ctx *resttypes.Context) *resttypes.APIError {
	return nil
}

func (m DefaultHandler) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	return nil, nil
}

func (m DefaultHandler) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	return nil, nil
}
