package adaptor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterHandler(router gin.IRoutes, handler http.Handler, urlMethods map[string][]string) {
	handlerFunc := gin.WrapH(handler)
	for url, methods := range urlMethods {
		for _, method := range methods {
			switch method {
			case http.MethodPost:
				router.POST(url, handlerFunc)
			case http.MethodDelete:
				router.DELETE(url, handlerFunc)
			case http.MethodPut:
				router.PUT(url, handlerFunc)
			case http.MethodGet:
				router.GET(url, handlerFunc)
			}
		}
	}
}
