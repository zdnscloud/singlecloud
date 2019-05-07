package server

import (
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/handler"
)

type Server struct {
	router *gin.Engine
}

func NewServer() (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(static.Serve("/", static.LocalFile("/www", false)))
	router.NoRoute(func(c *gin.Context) {
		c.File("/www/index.html")
	})

	app := handler.NewApp()
	if err := app.RegisterHandler(router); err != nil {
		log.Fatalf("register handler failed:%s", err.Error())
	}

	agent := clusteragent.New()
	agent.RegisterAgentHandler(router)

	return &Server{
		router: router,
	}, nil
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
