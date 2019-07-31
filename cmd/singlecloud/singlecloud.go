package main

import (
	"flag"
	"fmt"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/globaldns"
	"github.com/zdnscloud/singlecloud/pkg/handler"
	"github.com/zdnscloud/singlecloud/pkg/k8seventwatcher"
	"github.com/zdnscloud/singlecloud/pkg/k8sshell"
	"github.com/zdnscloud/singlecloud/server"
	"github.com/zdnscloud/singlecloud/storage"
)

const EventBufLen = 1000

var (
	version string
	build   string
)

func main() {
	var addr string
	var casServer string
	var globaldnsAddr string
	var showVersion bool
	var dbFilePath string
	flag.StringVar(&addr, "listen", ":80", "server listen address")
	flag.StringVar(&globaldnsAddr, "dns", "", "globaldns cmd server listen address")
	flag.StringVar(&casServer, "cas", "", "cas server address")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.StringVar(&dbFilePath, "db", "", "db file path")
	flag.Parse()

	if showVersion {
		fmt.Printf("singlecloud %s (build at %s)\n", version, build)
		return
	}

	log.InitLogger(log.Debug)
	eventBus := pubsub.New(EventBufLen)

	db, err := storage.New(dbFilePath)
	if err != nil {
		log.Fatalf("init database failed: %s", err.Error())
	}
	defer db.Close()

	if globaldnsAddr != "" {
		if err := globaldns.New(globaldnsAddr, eventBus); err != nil {
			log.Fatalf("create globaldns failed: %v", err.Error())
		}
	}

	authenticator, err := authentication.New(casServer, db)
	if err != nil {
		log.Fatalf("create authenticator failed:%s", err.Error())
	}

	authorizer, err := authorization.New(db)
	if err != nil {
		log.Fatalf("create authorizer failed:%s", err.Error())
	}

	app := handler.NewApp(authenticator, authorizer, eventBus, db)

	server, err := server.NewServer(authenticator.MiddlewareFunc())
	if err != nil {
		log.Fatalf("create server failed:%s", err.Error())
	}

	if err := server.RegisterHandler(authenticator); err != nil {
		log.Fatalf("register redirect handler failed:%s", err.Error())
	}

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
