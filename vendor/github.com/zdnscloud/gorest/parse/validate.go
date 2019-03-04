package parse

import (
	"fmt"
	"net/http"

	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/gorest/util/slice"
)

var (
	supportedMethods = map[string]bool{
		http.MethodPost:   true,
		http.MethodGet:    true,
		http.MethodPut:    true,
		http.MethodDelete: true,
	}
)

func ValidateMethod(request *types.APIContext) *types.APIError {
	if !supportedMethods[request.Method] {
		return types.NewAPIError(types.MethodNotAllowed, fmt.Sprintf("Method %s not supported", request.Method))
	}

	if request.Type == "" || request.Schema == nil {
		return types.NewAPIError(types.NotFound, "no found schema")
	}

	allowed := request.Schema.ResourceMethods
	if request.ID == "" {
		allowed = request.Schema.CollectionMethods
	}

	if slice.ContainsString(allowed, request.Method) {
		return nil
	}

	return types.NewAPIError(types.MethodNotAllowed, fmt.Sprintf("Method %s not supported", request.Method))
}
