package server

import (
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest/adaptor"
)

const ListenAddr = "0.0.0.0:80"

type Server struct {
	router *gin.Engine
}

func NewServer() (*Server, error) {
	restServer, err := newRestServer()
	if err != nil {
		return nil, err
	}
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(static.Serve("/", static.LocalFile("/www", false)))
	adaptor.RegisterHandler(router, gin.WrapH(restServer.server), restServer.server.Schemas.UrlMethods())

	return &Server{
		router: router,
	}, nil
}

func (s *Server) Run() {
	err := s.router.Run(ListenAddr)
	if err != nil {
		panic("listen " + ListenAddr + " fatal:" + err.Error())
	}
}
