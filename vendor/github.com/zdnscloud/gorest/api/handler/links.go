package handler

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/gorest/util"
)

func addLinks(ctx *types.Context, schema *types.Schema, obj types.Object) {
	links := make(map[string]string)
	self := genResourceLink(ctx.Request, obj.GetID())
	if util.ContainsString(schema.ResourceMethods, http.MethodGet) {
		links["self"] = self
	}

	if util.ContainsString(schema.ResourceMethods, http.MethodPut) {
		links["update"] = self
	}

	if util.ContainsString(schema.ResourceMethods, http.MethodDelete) {
		links["remove"] = self
	}

	if util.ContainsString(schema.CollectionMethods, http.MethodGet) {
		links["collection"] = genCollectionLink(ctx.Request, obj.GetID())
	}

	for _, childPluralName := range ctx.Schemas.GetChildren(obj.GetType()) {
		links[childPluralName] = genChildLink(ctx.Request, obj.GetID(), childPluralName)
	}

	obj.SetLinks(links)
}

func addResourceLinks(ctx *types.Context, obj interface{}) {
	if object, ok := obj.(types.Object); ok {
		addLinks(ctx, ctx.Object.GetSchema(), object)
	}
}

func addCollectionLinks(ctx *types.Context, collection *types.Collection) {
	collection.Links = map[string]string{
		"self": getRequestURL(ctx.Request),
	}

	sliceData := reflect.ValueOf(collection.Data)
	if sliceData.Kind() == reflect.Slice {
		for i := 0; i < sliceData.Len(); i++ {
			addResourceLinks(ctx, sliceData.Index(i).Interface())
		}
	}
}

func genResourceLink(req *http.Request, id string) string {
	if id == "" {
		return ""
	}

	requestURL := getRequestURL(req)
	if strings.HasSuffix(requestURL, "/"+id) {
		return requestURL
	} else {
		return requestURL + "/" + id
	}
}

func genCollectionLink(req *http.Request, id string) string {
	requestURL := getRequestURL(req)
	if id != "" && strings.HasSuffix(requestURL, "/"+id) {
		index := strings.LastIndex(requestURL, id)
		return requestURL[:index-1]
	}

	return requestURL
}

func genChildLink(req *http.Request, id, childPluralName string) string {
	if id == "" {
		return ""
	}

	return genResourceLink(req, id) + "/" + childPluralName
}

func getRequestURL(req *http.Request) string {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, req.URL.Path)
}
