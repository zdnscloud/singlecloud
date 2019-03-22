package server

import (
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/singlecloud/pkg/handler"
)

type Server struct {
	router *gin.Engine
}

func NewServer() (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	app := handler.NewApp()
	router := gin.New()
	router.Use(static.Serve("/", static.LocalFile("/www", false)))
	router.NoRoute(func(c *gin.Context) {
		c.File("/www/index.html")
	})

	if err := app.RegisterHandler(router); err != nil {
		panic("register handler failed:" + err.Error())
	}

	return &Server{
		router: router,
	}, nil
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
