package server

import (
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router *gin.Engine
}

type WebHandler interface {
	RegisterHandler(gin.IRoutes) error
}

func NewServer(middlewares ...gin.HandlerFunc) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middlewares...)
	router.Use(static.Serve("/assets", static.LocalFile("/www", false)))
	router.NoRoute(func(c *gin.Context) {
		c.File("/www/index.html")
	})
	return &Server{
		router: router,
	}, nil
}

func (s *Server) RegisterHandler(h WebHandler) error {
	return h.RegisterHandler(s.router)
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
