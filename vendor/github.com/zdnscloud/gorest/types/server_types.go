package types

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

type RawResource struct {
	ID           string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Type         string                 `json:"type,omitempty" yaml:"type,omitempty"`
	Schema       *Schema                `json:"-" yaml:"-"`
	Actions      map[string]string      `json:"actions,omitempty" yaml:"actions,omitempty"`
	Values       map[string]interface{} `json:",inline" yaml:",inline"`
	DropReadOnly bool                   `json:"-" yaml:"-"`
}

func (r *RawResource) AddAction(apiContext *APIContext, name string) {
	r.Actions[name] = apiContext.URLBuilder.Action(name, r)
}

func (r *RawResource) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.ToMap())
}

func (r *RawResource) ToMap() map[string]interface{} {
	data := map[string]interface{}{}
	for k, v := range r.Values {
		data[k] = v
	}

	if r.ID != "" && !r.DropReadOnly {
		data["id"] = r.ID
	}

	if r.Type != "" && !r.DropReadOnly {
		data["type"] = r.Type
	}
	if r.Schema.BaseType != "" && !r.DropReadOnly {
		data["baseType"] = r.Schema.BaseType
	}

	if len(r.Actions) > 0 && !r.DropReadOnly {
		data["actions"] = r.Actions
	}
	return data
}

type ActionHandler func(actionName string, action *Action, request *APIContext) error

type RequestHandler func(request *APIContext, next RequestHandler) error

type CollectionFormatter func(request *APIContext, collection *GenericCollection)

type ErrorHandler func(request *APIContext, err error)

type ResponseWriter interface {
	Write(apiContext *APIContext, code int, obj interface{})
}

type AccessControl interface {
	CanCreate(apiContext *APIContext, schema *Schema) error
	CanList(apiContext *APIContext, schema *Schema) error
	CanGet(apiContext *APIContext, schema *Schema) error
	CanUpdate(apiContext *APIContext, schema *Schema) error
	CanDelete(apiContext *APIContext, schema *Schema) error
}

type APIContext struct {
	Action         string
	ID             string
	Type           string
	Method         string
	Schema         *Schema
	Schemas        *Schemas
	Version        *APIVersion
	SchemasVersion *APIVersion
	Query          url.Values
	ResponseFormat string
	ResponseWriter ResponseWriter
	URLBuilder     URLBuilder
	AccessControl  AccessControl
	SubContext     map[string]string
	Request        *http.Request
	Response       http.ResponseWriter
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

func GetAPIContext(ctx context.Context) *APIContext {
	apiContext, _ := ctx.Value(apiContextKey{}).(*APIContext)
	return apiContext
}

func (r *APIContext) Option(key string) string {
	return r.Query.Get("_" + key)
}

func (r *APIContext) WriteResponse(code int, obj interface{}) {
	r.ResponseWriter.Write(r, code, obj)
}

type URLBuilder interface {
	Current() string
	Collection(schema *Schema, versionOverride *APIVersion) string
	CollectionAction(schema *Schema, versionOverride *APIVersion, action string) string
	RelativeToRoot(path string) string
	Version(version APIVersion) string
	Marker(marker string) string
	SetSubContext(subContext string)
	Action(action string, resource *RawResource) string
}
