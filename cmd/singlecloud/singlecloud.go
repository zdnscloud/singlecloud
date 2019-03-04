package main

import (
	"flag"

	"github.com/zdnscloud/singlecloud/pkg/api/server"
	"github.com/zdnscloud/singlecloud/pkg/logger"
)

func main() {
	flag.Parse()
	if err := logger.InitLogger(); err != nil {
		panic("init logger failed:" + err.Error())
	}

	server, err := server.NewServer()
	if err != nil {
		panic("create server failed:" + err.Error())
	}

	server.Run()
}