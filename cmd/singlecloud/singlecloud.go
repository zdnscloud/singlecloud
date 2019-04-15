package main

import (
	"flag"
	"fmt"

	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/server"
)

var (
	version string
	build   string
)

func main() {
	var addr string
	var showVersion bool
	flag.StringVar(&addr, "listen", ":80", "server listen address")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

	if showVersion {
		fmt.Printf("singlecloud %s (build at %s)\n", version, build)
		return
	}

	logger.InitLogger()

	server, err := server.NewServer()
	if err != nil {
		logger.Fatal("create server failed:%s", err.Error())
	}

	if err := server.Run(addr); err != nil {
		logger.Fatal("server run failed:%s", err.Error())
	}
}
