package server

import (
	"path"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/singlecloud/config"
)

var (
	crtFile = "server.crt"
	keyFile = "server.key"
)

type Server struct {
	router *gin.Engine
	addr   string
	crt    string
	key    string
}

func NewServer(conf *config.SingleCloudConf) (*Server, error) {
	restServer, err := newRestServer()
	if err != nil {
		return nil, err
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	adaptor.RegisterHandler(router, restServer.server)
	return &Server{
		router: router,
		addr:   conf.Server.Addr,
		crt:    path.Join(conf.Server.AuthDir, crtFile),
		key:    path.Join(conf.Server.AuthDir, keyFile),
	}, nil
}

func (s *Server) Run() {
	s.router.Run(s.addr)
}
