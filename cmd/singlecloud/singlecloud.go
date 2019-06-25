package main

import (
	"flag"
	"fmt"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/globaldns"
	"github.com/zdnscloud/singlecloud/pkg/handler"
	"github.com/zdnscloud/singlecloud/pkg/k8seventwatcher"
	"github.com/zdnscloud/singlecloud/pkg/k8sshell"
	"github.com/zdnscloud/singlecloud/pkg/model"
	"github.com/zdnscloud/singlecloud/server"
)

const EventBufLen = 1000

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
	eventBus := pubsub.New(EventBufLen)

	if err := model.InitResourceStore(); err != nil {
		log.Fatalf("init database failed: %s", err.Error())
	}

	if globaldnsAddr != "" {
		if err := globaldns.New(globaldnsAddr, eventBus); err != nil {
			log.Fatalf("create globaldns failed: %v", err.Error())
		}
	}

	server, err := server.NewServer()
	if err != nil {
		log.Fatalf("create server failed:%s", err.Error())
	}

	app := handler.NewApp(eventBus)
	if err := server.RegisterHandler(app); err != nil {
		log.Fatalf("register resource handler failed:%s", err.Error())
	}

	watcher := k8seventwatcher.New(eventBus)
	if err := server.RegisterHandler(watcher); err != nil {
		log.Fatalf("register k8s event watcher failed:%s", err.Error())
	}

	agent := clusteragent.New()
	if err := server.RegisterHandler(agent); err != nil {
		log.Fatalf("register agent failed:%s", err.Error())
	}

	shellExecutor := k8sshell.New(eventBus)
	if err := server.RegisterHandler(shellExecutor); err != nil {
		log.Fatalf("register shell executor failed:%s", err.Error())
	}

	if err := server.Run(addr); err != nil {
		log.Fatalf("server run failed:%s", err.Error())
	}
}
