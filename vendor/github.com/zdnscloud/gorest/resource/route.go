package resource

import (
	"net/http"
)

type HttpMethod string

var SupportedMethods = []HttpMethod{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPost}

type ResourceRoute map[HttpMethod][]string

func NewResourceRoute() ResourceRoute {
	return make(map[HttpMethod][]string)
}

func (a ResourceRoute) Merge(b ResourceRoute) ResourceRoute {
	for _, method := range SupportedMethods {
		a[method] = append(a[method], b[method]...)
	}
	return a
}

func (a ResourceRoute) AddPathForMethod(method HttpMethod, path string) {
	a[method] = append(a[method], path)
}
