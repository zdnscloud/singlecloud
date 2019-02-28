package server

import (
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
	adaptor.RegisterHandler(router, restServer.server)
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
