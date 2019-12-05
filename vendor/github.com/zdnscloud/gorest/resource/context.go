package resource

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/zdnscloud/gorest/error"
)

const (
	Eq      Modifier = "eq"
	Ne      Modifier = "ne"
	Lt      Modifier = "lt"
	Gt      Modifier = "gt"
	Lte     Modifier = "lte"
	Gte     Modifier = "gte"
	Prefix  Modifier = "prefix"
	Suffix  Modifier = "suffix"
	Like    Modifier = "like"
	NotLike Modifier = "notlike"
	Null    Modifier = "null"
	NotNull Modifier = "notnull"
)

type Context struct {
	Schemas  SchemaManager
	Request  *http.Request
	Response http.ResponseWriter
	Resource Resource
	Method   string
	params   map[string]interface{}
	filters  []Filter
}

type Filter struct {
	Name string
	Modifier
	Value []string
}

type Modifier string

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
		filters:  genFilters(req.URL),
	}, nil
}

func (ctx *Context) Set(key string, value interface{}) {
	ctx.params[key] = value
}

func (ctx *Context) Get(key string) (interface{}, bool) {
	v, ok := ctx.params[key]
	return v, ok
}

func (ctx *Context) GetFilters() []Filter {
	return ctx.filters
}

func genFilters(url *url.URL) []Filter {
	filters := make([]Filter, 0)
	for k, v := range url.Query() {
		var filter Filter
		i := strings.LastIndexAny(k, "_")
		if i < 0 {
			filter.Name = k
			filter.Modifier = Eq
		} else {
			filter.Name = k[:i]
			filter.Modifier = VerifyModifier(k[i+1:])
		}
		filter.Value = v
		filters = append(filters, filter)
	}
	return filters
}

func VerifyModifier(str string) Modifier {
	switch str {
	case "eq":
		return Eq
	case "ne":
		return Ne
	case "lt":
		return Lt
	case "gt":
		return Gt
	case "lte":
		return Lte
	case "gte":
		return Gte
	case "prefix":
		return Prefix
	case "suffix":
		return Suffix
	case "like":
		return Like
	case "notlike":
		return NotLike
	case "null":
		return Null
	case "notnull":
		return NotNull
	default:
		return Modifier(fmt.Sprintf("Unkown(%s)", str))
	}
}
