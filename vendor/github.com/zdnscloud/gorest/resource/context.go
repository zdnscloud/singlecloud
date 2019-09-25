package resource

import (
	"github.com/zdnscloud/gorest/error"
	"net/http"
)

type Context struct {
	Schemas  SchemaManager
	Request  *http.Request
	Response http.ResponseWriter
	Resource Resource
	Method   string
	params   map[string]interface{}
}

func NewContext(resp http.ResponseWriter, req *http.Request, schemas SchemaManager) (*Context, *error.APIError) {
	r, err := schemas.CreateResourceFromRequest(req)
	if err != nil {
		return nil, err
	}

	return &Context{
		Request:  req,
		Response: resp,
		Resource: r,
		Schemas:  schemas,
		Method:   req.Method,
		params:   make(map[string]interface{}),
	}, nil
}

func (ctx *Context) Set(key string, value interface{}) {
	ctx.params[key] = value
}

func (ctx *Context) Get(key string) (interface{}, bool) {
	v, ok := ctx.params[key]
	return v, ok
}
