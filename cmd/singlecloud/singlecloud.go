package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/kvzoo/client"
	dbserver "github.com/zdnscloud/kvzoo/server"
	"github.com/zdnscloud/singlecloud/config"
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
	configFile  string
	version     string
	showVersion bool
	genConfFile bool
	build       string
)

func main() {
	flag.StringVar(&configFile, "c", "singlecloud.conf", "configure file path")
	flag.BoolVar(&genConfFile, "gen", false, "generate initial configure file to current directory")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

	log.InitLogger(log.Debug)

	if showVersion {
		log.Infof("singlecloud %s (build at %s)\n", version, build)
		return
	}

	if genConfFile {
		if err := genInitConfig(); err != nil {
			log.Fatalf("generate initial configure file failed:%s", err.Error())
		}
		return
	}

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load configure file failed:%s", err.Error())
	}

	if conf.DB.Role == config.Master {
		runAsMaster(conf)
	} else {
		runAsSlave(conf)
	}
}

func runAsMaster(conf *config.SinglecloudConf) {
	eventBus := pubsub.New(EventBufLen)

	stopCh := make(chan struct{})
	dbClient, err := initMasterDB(conf.DB.Port, conf.DB.Path, conf.DB.SlaveDBAddr, stopCh)
	if err != nil {
		log.Fatalf("create database failed: %s", err.Error())
	}
	defer close(stopCh)

	if conf.Server.DNSAddr != "" {
		if err := globaldns.New(conf.Server.DNSAddr, eventBus); err != nil {
			log.Fatalf("create globaldns failed: %v", err.Error())
		}
	}

	authenticator, err := authentication.New(conf.Server.CasAddr, dbClient)
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
	app, err := handler.NewApp(authenticator, authorizer, eventBus, agent, dbClient, conf.Chart.Path, version, conf.Chart.Repo, conf.Registry)
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

	if conf.DB.SlaveDBAddr != "" {
		if _, err := dbClient.Checksum(); err != nil {
			log.Fatalf("db isn't in sync:%s", err.Error())
		}
	}

	if err := server.Run(conf.Server.Addr); err != nil {
		log.Fatalf("server run failed:%s", err.Error())
	}
}

func initMasterDB(localDBPort int, dbFilePath, slaveDBAddress string, stopCh chan struct{}) (kvzoo.DB, error) {
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
	if slaveDBAddress != "" {
		slaves = append(slaves, slaveDBAddress)
	}
	dbClient, err := client.New(dbServerAddr, slaves)
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

func runAsSlave(conf *config.SinglecloudConf) {
	dbServerAddr := fmt.Sprintf(":%d", conf.DB.Port)
	db, err := dbserver.NewWithBoltDB(dbServerAddr, path.Join(conf.DB.Path, DBFileName))
	if err != nil {
		log.Fatalf("start slave failed:%s", err.Error())
		return
	}
	db.Start()
}

func genInitConfig() error {
	yamlConfig, err := yaml.Marshal(config.CreateDefaultConfig())
	if err != nil {
		return err
	}
	configFile := "./singlecloud.conf"
	log.Debugf("Deploying cluster configuration file: %s", configFile)
	return ioutil.WriteFile(configFile, yamlConfig, 0640)
}
