package api

import "github.com/zdnscloud/gorest/types"

type ResponseWriter interface {
	Write(apiContext *types.APIContext, code int, obj interface{})
}
