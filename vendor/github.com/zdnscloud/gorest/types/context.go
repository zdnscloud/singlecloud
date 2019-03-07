package types

import (
	"context"
	"net/http"
	"net/url"
)

type APIContext struct {
	Action         string
	ID             string
	Type           string
	Method         string
	Schema         *Schema
	Schemas        *Schemas
	Version        *APIVersion
	Query          url.Values
	ResponseFormat string
	Request        *http.Request
	Response       http.ResponseWriter
	Parent         Object
}

type apiContextKey struct{}

func NewAPIContext(req *http.Request, resp http.ResponseWriter, schemas *Schemas) *APIContext {
	apiCtx := &APIContext{
		Response: resp,
		Schemas:  schemas,
	}
	ctx := context.WithValue(req.Context(), apiContextKey{}, apiCtx)
	apiCtx.Request = req.WithContext(ctx)
	return apiCtx
}
