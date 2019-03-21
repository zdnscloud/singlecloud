package handler

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/gorest/util"
)

func addLinks(apiContext *types.APIContext, obj types.Object) {
	links := make(map[string]string)
	self := genResourceLink(apiContext.Request, obj.GetID())
	links["self"] = self

	if util.ContainsString(apiContext.Schema.ResourceMethods, http.MethodPut) {
		links["update"] = self
	}

	if util.ContainsString(apiContext.Schema.ResourceMethods, http.MethodDelete) {
		links["remove"] = self
	}

	if util.ContainsString(apiContext.Schema.CollectionMethods, http.MethodGet) {
		links["collection"] = genCollectionLink(apiContext.Request, obj.GetID())
	}

	for _, childPluralName := range apiContext.Schemas.GetChildren(apiContext.Schema.ID) {
		links[childPluralName] = genChildLink(apiContext.Request, obj.GetID(), childPluralName)
	}

	obj.SetLinks(links)
}

func addResourceLinks(apiContext *types.APIContext, obj interface{}) {
	if object, ok := obj.(types.Object); ok {
		addLinks(apiContext, object)
	}
}

func addCollectionLinks(apiContext *types.APIContext, collection *types.Collection) {
	collection.Links = map[string]string{
		"self": getRequestURL(apiContext.Request),
	}

	sliceData := reflect.ValueOf(collection.Data)
	if sliceData.Kind() == reflect.Slice {
		for i := 0; i < sliceData.Len(); i++ {
			addResourceLinks(apiContext, sliceData.Index(i).Interface())
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
