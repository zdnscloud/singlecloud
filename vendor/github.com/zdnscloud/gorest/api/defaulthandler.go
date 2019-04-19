package api

import (
	"github.com/zdnscloud/gorest/types"
)

var _ types.Handler = DefaultHandler{}

type DefaultHandler struct{}

func (m DefaultHandler) Create(ctx *types.Context, yamlConf []byte) (interface{}, *types.APIError) {
	return nil, nil
}

func (m DefaultHandler) List(ctx *types.Context) interface{} {
	return nil
}

func (m DefaultHandler) Get(ctx *types.Context) interface{} {
	return nil
}

func (m DefaultHandler) Delete(ctx *types.Context) *types.APIError {
	return nil
}

func (m DefaultHandler) Update(ctx *types.Context) (interface{}, *types.APIError) {
	return nil, nil
}

func (m DefaultHandler) Action(ctx *types.Context) (interface{}, *types.APIError) {
	return nil, nil
}
