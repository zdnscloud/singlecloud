package handler

import (
	"encoding/json"

	"github.com/zdnscloud/gorest/types"
	yaml "gopkg.in/yaml.v2"
)

func WriteResponse(apiContext *types.APIContext, status int, result interface{}) {
	resp := apiContext.Response
	resp.WriteHeader(status)
	var body []byte
	switch apiContext.ResponseFormat {
	case "json":
		resp.Header().Set("content-type", "application/json")
		body, _ = json.Marshal(result)
	case "yaml":
		resp.Header().Set("content-type", "application/yaml")
		body, _ = yaml.Marshal(result)
	}
	resp.Write(body)
}
