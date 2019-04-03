package handler

import (
	"encoding/json"

	"github.com/zdnscloud/gorest/types"
	yaml "gopkg.in/yaml.v2"
)

func WriteResponse(ctx *types.Context, status int, result interface{}) {
	resp := ctx.Response
	resp.WriteHeader(status)
	var body []byte
	switch ctx.ResponseFormat {
	case types.ResponseJSON:
		resp.Header().Set("content-type", "application/json")
		body, _ = json.Marshal(result)
	case types.ResponseYAML:
		resp.Header().Set("content-type", "application/yaml")
		body, _ = yaml.Marshal(result)
	}
	resp.Write(body)
}
