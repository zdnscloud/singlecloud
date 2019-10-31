package main

import (
	"flag"
	"fmt"
	"path"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/kvzoo/client"
	dbserver "github.com/zdnscloud/kvzoo/server"
	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/globaldns"
	"github.com/zdnscloud/singlecloud/pkg/handler"
	"github.com/zdnscloud/singlecloud/pkg/k8seventwatcher"
	"github.com/zdnscloud/singlecloud/pkg/k8sshell"
	"github.com/zdnscloud/singlecloud/server"
)

const (
	EventBufLen = 1000
	DBFileName  = "singlecloud.db"
)

var (
	version string
	build   string
)

func main() {
	var (
		addr               string
		casServer          string
		globaldnsAddr      string
		showVersion        bool
		chartDir           string
		dbFilePath         string
		dbPort             int
		secondaryDBAddress string
		repoUrl            string
	)

	flag.StringVar(&addr, "listen", ":80", "server listen address")
	flag.StringVar(&globaldnsAddr, "dns", "", "globaldns cmd server listen address")
	flag.StringVar(&casServer, "cas", "", "cas server address")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.StringVar(&chartDir, "chart", "", "chart path")
	flag.StringVar(&dbFilePath, "db", "", "db file path")
	flag.IntVar(&dbPort, "dbport", 6666, "db server port")
	flag.StringVar(&secondaryDBAddress, "secondary-addr", "", "secondary db address")
	flag.StringVar(&repoUrl, "repo", "", "chart repo url")
	flag.Parse()

	if showVersion {
		fmt.Printf("singlecloud %s (build at %s)\n", version, build)
		return
	}

	log.InitLogger(log.Debug)
	eventBus := pubsub.New(EventBufLen)

	stopCh := make(chan struct{})
	dbClient, err := initDB(dbPort, dbFilePath, secondaryDBAddress, stopCh)
	if err != nil {
		log.Fatalf("create database failed: %s", err.Error())
	}
	defer close(stopCh)

	if globaldnsAddr != "" {
		if err := globaldns.New(globaldnsAddr, eventBus); err != nil {
			log.Fatalf("create globaldns failed: %v", err.Error())
		}
	}

	authenticator, err := authentication.New(casServer, dbClient)
	if err != nil {
		log.Fatalf("create authenticator failed:%s", err.Error())
	}

	authorizer, err := authorization.New(dbClient)
	if err != nil {
		log.Fatalf("create authorizer failed:%s", err.Error())
	}

	server, err := server.NewServer(authenticator.MiddlewareFunc())
	if err != nil {
		log.Fatalf("create server failed:%s", err.Error())
	}

	agent := clusteragent.New()
	app, err := handler.NewApp(authenticator, authorizer, eventBus, agent, dbClient, chartDir, version, repoUrl)
	if err != nil {
		log.Fatalf("create app failed %s", err.Error())
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

func initDB(localDBPort int, dbFilePath, secondaryDBAddress string, stopCh chan struct{}) (kvzoo.DB, error) {
	dbServerAddr := fmt.Sprintf(":%d", localDBPort)
	db, err := dbserver.NewWithBoltDB(dbServerAddr, path.Join(dbFilePath, DBFileName))
	if err != nil {
		return nil, err
	}
	dbStarted := make(chan struct{})
	go func() {
		close(dbStarted)
		db.Start()
	}()
	<-dbStarted

	var slaves []string
	if secondaryDBAddress != "" {
		slaves = append(slaves, secondaryDBAddress)
	}
	dbClient, err := client.New(dbServerAddr, nil)
	if err != nil {
		db.Stop()
		return nil, err
	}

	go func() {
		<-stopCh
		db.Stop()
	}()
	return dbClient, nil
}
