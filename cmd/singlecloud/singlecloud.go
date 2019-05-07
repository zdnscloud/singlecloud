package main

import (
	"flag"
	"fmt"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/pkg/globaldns"
	"github.com/zdnscloud/singlecloud/server"
)

var (
	version string
	build   string
)

func main() {
	var addr string
	var globaldnsAddr string
	var showVersion bool
	flag.StringVar(&addr, "listen", ":80", "server listen address")
	flag.StringVar(&globaldnsAddr, "dns", "", "globaldns cmd server listen address")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

	if showVersion {
		fmt.Printf("singlecloud %s (build at %s)\n", version, build)
		return
	}

	log.InitLogger(log.Debug)

	if globaldnsAddr != "" {
		err := globaldns.Init(globaldnsAddr)
		if err != nil {
			log.Fatalf("init globaldns failed: %v", err.Error())
		}
	}

	server, err := server.NewServer()
	if err != nil {
		log.Fatalf("create server failed:%s", err.Error())
	}

	if err := server.Run(addr); err != nil {
		log.Fatalf("server run failed:%s", err.Error())
	}
}
