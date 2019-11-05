package resource

import (
	goresterr "github.com/zdnscloud/gorest/error"
	"net/http"
)

type SchemaManager interface {
	Import(*APIVersion, ResourceKind, interface{}) error

	//same with import, but will panic if get error
	MustImport(*APIVersion, ResourceKind, interface{})

	//for GET/ DELETE, return empty resource, with id and parent set,
	//for POST and PUT, the resource unmarshal from body will be returned
	//also support default value and validation check
	CreateResourceFromRequest(*http.Request) (Resource, *goresterr.APIError)

	//based on handler to generate route for the resources
	GenerateResourceRoute() ResourceRoute
	WriteJsonDocs(v *APIVersion, path string) error
}

type Schema interface {
	GetHandler() Handler
	AddLinksToResource(r Resource, httpSchemeAndHost string) error
	AddLinksToResourceCollection(rs *ResourceCollection, httpSchemeAndHost string) error
	WriteJsonDoc(path string, parents []string) error
}
