package server

import (
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest/adaptor"

	"github.com/zdnscloud/singlecloud/pkg/k8smanager"
)

type Server struct {
	router *gin.Engine
}

func NewServer() (*Server, error) {
	restHandler, wsHandler, err := k8smanager.NewRestHandler()
	if err != nil {
		return nil, err
	}
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(static.Serve("/", static.LocalFile("/www", false)))
	router.NoRoute(func(c *gin.Context) {
		c.File("/www/index.html")
	})

	router.GET("/zcloud/ws/clusters/:id", func(c *gin.Context) {
		wsHandler.OpenConsole(c.Param("id"), c.Request, c.Writer)
	})
	adaptor.RegisterHandler(router, gin.WrapH(restHandler), restHandler.Schemas.UrlMethods())

	return &Server{
		router: router,
	}, nil
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
