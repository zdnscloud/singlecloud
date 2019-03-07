package handler

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/gorest/util/slice"
)

func addLinks(apiContext *types.APIContext, obj types.Object) {
	links := make(map[string]string)
	self := genResourceLink(apiContext.Request, obj.GetID())
	links["self"] = self

	if slice.ContainsString(apiContext.Schema.ResourceMethods, http.MethodPut) {
		links["update"] = self
	}

	if slice.ContainsString(apiContext.Schema.ResourceMethods, http.MethodDelete) {
		links["remove"] = self
	}

	if slice.ContainsString(apiContext.Schema.CollectionMethods, http.MethodGet) {
		links["collection"] = genCollectionLink(apiContext.Request, obj.GetID())
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
	index := strings.LastIndex(requestURL, id)
	if index == -1 {
		return requestURL
	} else {
		return requestURL[:index-1]
	}
}

func getRequestURL(req *http.Request) string {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, req.URL.Path)
}
