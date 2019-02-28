package api

import (
	"net/http"

	"github.com/zdnscloud/gorest/api/handler"
	"github.com/zdnscloud/gorest/api/writer"
	"github.com/zdnscloud/gorest/authorization"
	"github.com/zdnscloud/gorest/httperror"
	ehandler "github.com/zdnscloud/gorest/httperror/handler"
	"github.com/zdnscloud/gorest/parse"
	"github.com/zdnscloud/gorest/types"
)

type Parser func(rw http.ResponseWriter, req *http.Request) (*types.APIContext, error)

type Server struct {
	Parser          Parser
	Resolver        parse.ResolverFunc
	ResponseWriters map[string]ResponseWriter
	Schemas         *types.Schemas
	URLParser       parse.URLParser
	Defaults        Defaults
	AccessControl   types.AccessControl
}

type Defaults struct {
	ActionHandler types.ActionHandler
	ListHandler   types.RequestHandler
	CreateHandler types.RequestHandler
	DeleteHandler types.RequestHandler
	UpdateHandler types.RequestHandler
	ErrorHandler  types.ErrorHandler
}

func NewAPIServer() *Server {
	s := &Server{
		Schemas: types.NewSchemas(),
		ResponseWriters: map[string]ResponseWriter{
			"json": &writer.EncodingResponseWriter{
				ContentType: "application/json",
				Encoder:     types.JSONEncoder,
			},
			"html": &writer.HTMLResponseWriter{
				EncodingResponseWriter: writer.EncodingResponseWriter{
					Encoder:     types.JSONEncoder,
					ContentType: "application/json",
				},
			},
			"yaml": &writer.EncodingResponseWriter{
				ContentType: "application/yaml",
				Encoder:     types.YAMLEncoder,
			},
		},
		Resolver:      parse.DefaultResolver,
		AccessControl: &authorization.AllAccess{},
		Defaults: Defaults{
			CreateHandler: handler.CreateHandler,
			DeleteHandler: handler.DeleteHandler,
			UpdateHandler: handler.UpdateHandler,
			ListHandler:   handler.ListHandler,
			ErrorHandler:  ehandler.ErrorHandler,
			ActionHandler: handler.ActionHandler,
		},
		URLParser: parse.DefaultURLParser,
	}

	s.Schemas.AddHook = s.setupDefaults
	s.Parser = s.parser
	return s
}

func (s *Server) parser(rw http.ResponseWriter, req *http.Request) (*types.APIContext, error) {
	ctx, err := parse.Parse(rw, req, s.Schemas, s.URLParser, s.Resolver)
	ctx.ResponseWriter = s.ResponseWriters[ctx.ResponseFormat]
	if ctx.ResponseWriter == nil {
		ctx.ResponseWriter = s.ResponseWriters["json"]
	}

	ctx.AccessControl = s.AccessControl

	return ctx, err
}

func (s *Server) AddSchemas(schemas *types.Schemas) error {
	if schemas.Err() != nil {
		return schemas.Err()
	}

	for _, schema := range schemas.Schemas() {
		s.Schemas.AddSchema(*schema)
	}

	return s.Schemas.Err()
}

func (s *Server) setupDefaults(schema *types.Schema) {
	if schema.ActionHandler == nil {
		schema.ActionHandler = s.Defaults.ActionHandler
	}

	if schema.ListHandler == nil {
		schema.ListHandler = s.Defaults.ListHandler
	}

	if schema.CreateHandler == nil {
		schema.CreateHandler = s.Defaults.CreateHandler
	}

	if schema.UpdateHandler == nil {
		schema.UpdateHandler = s.Defaults.UpdateHandler
	}

	if schema.DeleteHandler == nil {
		schema.DeleteHandler = s.Defaults.DeleteHandler
	}

	if schema.ErrorHandler == nil {
		schema.ErrorHandler = s.Defaults.ErrorHandler
	}
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if apiResponse, err := s.handle(rw, req); err != nil {
		s.handleError(apiResponse, err)
	}
}

func (s *Server) handle(rw http.ResponseWriter, req *http.Request) (*types.APIContext, error) {
	apiRequest, err := s.Parser(rw, req)
	if err != nil {
		return apiRequest, err
	}

	if err := CheckCSRF(apiRequest); err != nil {
		return apiRequest, err
	}

	action, err := ValidateAction(apiRequest)
	if err != nil {
		return apiRequest, err
	}

	if apiRequest.Schema == nil {
		return apiRequest, nil
	}

	if action == nil && apiRequest.Type != "" {
		var handler types.RequestHandler
		var nextHandler types.RequestHandler
		switch apiRequest.Method {
		case http.MethodGet:
			if apiRequest.ID == "" {
				if err := apiRequest.AccessControl.CanList(apiRequest, apiRequest.Schema); err != nil {
					return apiRequest, err
				}
			} else {
				if err := apiRequest.AccessControl.CanGet(apiRequest, apiRequest.Schema); err != nil {
					return apiRequest, err
				}
			}
			handler = apiRequest.Schema.ListHandler
			nextHandler = s.Defaults.ListHandler
		case http.MethodPost:
			if err := apiRequest.AccessControl.CanCreate(apiRequest, apiRequest.Schema); err != nil {
				return apiRequest, err
			}
			handler = apiRequest.Schema.CreateHandler
			nextHandler = s.Defaults.CreateHandler
		case http.MethodPut:
			if err := apiRequest.AccessControl.CanUpdate(apiRequest, apiRequest.Schema); err != nil {
				return apiRequest, err
			}
			handler = apiRequest.Schema.UpdateHandler
			nextHandler = s.Defaults.UpdateHandler
		case http.MethodDelete:
			if err := apiRequest.AccessControl.CanDelete(apiRequest, apiRequest.Schema); err != nil {
				return apiRequest, err
			}
			handler = apiRequest.Schema.DeleteHandler
			nextHandler = s.Defaults.DeleteHandler
		}

		if handler == nil {
			return apiRequest, httperror.NewAPIError(httperror.NotFound, "")
		}

		return apiRequest, handler(apiRequest, nextHandler)
	} else if action != nil {
		return apiRequest, apiRequest.Schema.ActionHandler(apiRequest.Action, action, apiRequest)
	}

	return apiRequest, nil
}

func (s *Server) handleError(apiRequest *types.APIContext, err error) {
	if apiRequest.Schema == nil {
		s.Defaults.ErrorHandler(apiRequest, err)
	} else if apiRequest.Schema.ErrorHandler != nil {
		apiRequest.Schema.ErrorHandler(apiRequest, err)
	}
}
