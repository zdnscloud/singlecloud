package adaptor

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest/resource"
)

func RegisterHandler(router gin.IRoutes, handler http.Handler, route resource.ResourceRoute) {
	handlerFunc := gin.WrapH(handler)
	for method, paths := range route {
		switch method {
		case http.MethodPost:
			for _, path := range paths {
				router.POST(path, handlerFunc)
			}
		case http.MethodDelete:
			for _, path := range paths {
				router.DELETE(path, handlerFunc)
			}
		case http.MethodPut:
			for _, path := range paths {
				router.PUT(path, handlerFunc)
			}
		case http.MethodGet:
			for _, path := range paths {
				router.GET(path, handlerFunc)
			}
		}
	}
}
