package main

import (
	"flag"
	"fmt"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/server"
)

var (
	version string
	build   string
)

func main() {
	var addr string
	var globaldns string
	var showVersion bool
	flag.StringVar(&addr, "listen", ":80", "server listen address")
	flag.StringVar(&globaldns, "dns", "", "globaldns cmd server listen address")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

	if showVersion {
		fmt.Printf("singlecloud %s (build at %s)\n", version, build)
		return
	}

	log.InitLogger(log.Debug)

	server, err := server.NewServer(globaldns)
	if err != nil {
		log.Fatalf("create server failed:%s", err.Error())
	}

	if err := server.Run(addr); err != nil {
		log.Fatalf("server run failed:%s", err.Error())
	}
}
