package adaptor

import (
	"net/http"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest/api"
	"github.com/zdnscloud/gorest/name"
)

func RegisterHandler(router gin.IRoutes, server *api.Server) {
	handlerFunc := gin.WrapH(server)
	for _, schema := range server.Schemas.Schemas() {
		url := path.Join("/"+schema.Version.Group, schema.Version.Path, schema.PluralName)
		if schema.Parent.Name != "" {
			url = path.Join("/"+schema.Version.Group, schema.Version.Path,
				name.GuessPluralName(schema.Parent.Name), ":"+schema.Parent.Name+"_id", schema.PluralName)
		}
		registerHandler(router, handlerFunc, url, schema.CollectionMethods)
		registerHandler(router, handlerFunc, path.Join(url, ":"+schema.ID+"_id"), schema.ResourceMethods)
	}
}

func registerHandler(router gin.IRoutes, handlerFunc gin.HandlerFunc, url string, methods []string) {
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
