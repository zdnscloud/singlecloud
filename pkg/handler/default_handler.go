package handler

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

type DefaultHandler struct {
}

func (m DefaultHandler) Create(obj resttypes.Object, yamlConf []byte) (interface{}, *resttypes.APIError) {
	return nil, nil
}

func (m DefaultHandler) List(obj resttypes.Object) interface{} {
	return nil
}

func (m DefaultHandler) Get(obj resttypes.Object) interface{} {
	return nil
}

func (m DefaultHandler) Delete(obj resttypes.Object) *resttypes.APIError {
	return nil
}

func (m DefaultHandler) Update(obj resttypes.Object) (interface{}, *resttypes.APIError) {
	return obj, nil
}

func (m DefaultHandler) Action(obj resttypes.Object, action string, params map[string]interface{}) (interface{}, *resttypes.APIError) {
	return params, nil
}
