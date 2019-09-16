package resource

import (
	goresterr "github.com/zdnscloud/gorest/error"
	"net/http"
)

type SchemaManager interface {
	Import(*APIVersion, ResourceKind, interface{}) error

	//for GET/ DELETE, return empty resource, with id and parent set,
	//for POST and PUT, the resource unmarshal from body will be returned
	//also support default value and validation check
	CreateResourceFromRequest(*http.Request) (Resource, *goresterr.APIError)

	//based on handler to generate route for the resources
	GenerateResourceRoute() ResourceRoute
}

type Schema interface {
	GetHandler() Handler
	GenerateLinks(r Resource, httpSchemeAndHost string) (map[ResourceLinkType]ResourceLink, error)
}
