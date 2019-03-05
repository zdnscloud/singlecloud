package main

import (
	"flag"

	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/server"
)

func main() {
	var addr string
	flag.StringVar(&addr, "listen", ":80", "server listen address")
	flag.Parse()

	if err := logger.InitLogger(); err != nil {
		panic("init logger failed:" + err.Error())
	}

	server, err := server.NewServer()
	if err != nil {
		panic("create server failed:" + err.Error())
	}

	if err := server.Run(addr); err != nil {
		panic("server run failed:" + err.Error())
	}
}
