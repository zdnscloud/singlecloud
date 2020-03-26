package server

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

type Server struct {
	router *gin.Engine
}

type WebHandler interface {
	RegisterHandler(gin.IRoutes) error
}

func NewServer(middlewares ...gin.HandlerFunc) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = os.Stdout
	router := gin.New()
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] client:%s \"%s %s\" %s %d %s %s\n",
			param.TimeStamp.Format(time.RFC3339),
			param.ClientIP,
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
		)
	}))
	router.Use(static.Serve("/assets/helm/icons", static.LocalFile("/helm-icons", false)))
	router.Use(static.Serve("/assets", static.LocalFile("/www", false)))
	router.Use(middlewares...)
	router.NoRoute(func(c *gin.Context) {
		c.File("/www/index.html")
	})

	p := ginprometheus.NewPrometheus("gin")
	p.Use(router)

	return &Server{
		router: router,
	}, nil
}

func (s *Server) RegisterHandler(h WebHandler) error {
	return h.RegisterHandler(s.router)
}

func (s *Server) Run(addr, certFile, keyFile string) error {
	return s.router.RunTLS(addr, certFile, keyFile)
}
