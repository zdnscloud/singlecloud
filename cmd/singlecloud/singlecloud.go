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
