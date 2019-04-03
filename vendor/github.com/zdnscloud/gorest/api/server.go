package api

import (
	"net/http"

	"github.com/zdnscloud/gorest/api/handler"
	"github.com/zdnscloud/gorest/parse"
	"github.com/zdnscloud/gorest/types"
)

type HandlerFunc func(*types.Context) *types.APIError
type HandlersChain []HandlerFunc

type Server struct {
	Schemas  *types.Schemas
	handlers HandlersChain
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
	ctx, err := parse.Parse(rw, req, s.Schemas)
	if err != nil {
		handler.WriteResponse(ctx, err.Status, err)
		return
	}

	for _, h := range s.handlers {
		if err := h(ctx); err != nil {
			handler.WriteResponse(ctx, err.Status, err)
			return
		}
	}
}

func (s *Server) Use(h HandlerFunc) {
	s.handlers = append(s.handlers, h)
}
