package handler

import (
	"encoding/json"

	"github.com/zdnscloud/gorest/types"
	yaml "gopkg.in/yaml.v2"
)

const ContentTypeKey = "Content-Type"

func WriteResponse(ctx *types.Context, status int, result interface{}) {
	resp := ctx.Response
	var body []byte
	switch ctx.ResponseFormat {
	case types.ResponseJSON:
		resp.Header().Set(ContentTypeKey, "application/json")
		body, _ = json.Marshal(result)
	case types.ResponseYAML:
		resp.Header().Set(ContentTypeKey, "application/yaml")
		body, _ = yaml.Marshal(result)
	}
	resp.WriteHeader(status)
	resp.Write(body)
}
