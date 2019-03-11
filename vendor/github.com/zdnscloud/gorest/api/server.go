package api

import (
	"net/http"

	"github.com/zdnscloud/gorest/api/handler"
	"github.com/zdnscloud/gorest/parse"
	"github.com/zdnscloud/gorest/types"
)

type Server struct {
	Schemas *types.Schemas
}

func NewAPIServer() *Server {
	s := &Server{
		Schemas: types.NewSchemas(),
	}

	return s
}

func (s *Server) AddSchemas(schemas *types.Schemas) error {
	if err := schemas.Err(); err != nil {
		return err
	}

	for _, schema := range schemas.Schemas() {
		s.Schemas.AddSchema(*schema)
	}

	return s.Schemas.Err()
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if apiResponse, err := s.handle(rw, req); err != nil {
		handler.WriteResponse(apiResponse, err.Status, err)
	}
}

func (s *Server) handle(rw http.ResponseWriter, req *http.Request) (*types.APIContext, *types.APIError) {
	apiRequest, err := parse.Parse(rw, req, s.Schemas)
	if err != nil {
		return apiRequest, err
	}

	action, err := ValidateAction(apiRequest)
	if err != nil {
		return apiRequest, err
	}

	if apiRequest.Schema == nil {
		return apiRequest, types.NewAPIError(types.NotFound, "no found schema")
	}

	if action == nil && apiRequest.Type != "" {
		var reqHandler types.RequestHandler
		switch apiRequest.Method {
		case http.MethodGet:
			reqHandler = handler.ListHandler
		case http.MethodPost:
			reqHandler = handler.CreateHandler
		case http.MethodPut:
			reqHandler = handler.UpdateHandler
		case http.MethodDelete:
			reqHandler = handler.DeleteHandler
		}

		if reqHandler == nil {
			return apiRequest, types.NewAPIError(types.NotFound, "no found request handler")
		}

		return apiRequest, reqHandler(apiRequest)
	} else if action != nil {
		return apiRequest, handler.ActionHandler(apiRequest, action)
	}

	return apiRequest, nil
}
