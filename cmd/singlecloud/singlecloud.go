package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/config"
	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/db"
	"github.com/zdnscloud/singlecloud/pkg/globaldns"
	"github.com/zdnscloud/singlecloud/pkg/handler"
	"github.com/zdnscloud/singlecloud/pkg/k8seventwatcher"
	"github.com/zdnscloud/singlecloud/pkg/k8sshell"
	"github.com/zdnscloud/singlecloud/server"
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
		fmt.Printf("singlecloud %s (build at %s)\n", version, build)
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
	stopCh := make(chan struct{})
	err := db.RunAsMaster(conf, stopCh)
	if err != nil {
		log.Fatalf("create database failed: %s", err.Error())
	}
	defer close(stopCh)

	if err := globaldns.New(conf.Server.DNSAddr); err != nil {
		log.Fatalf("create globaldns failed: %v", err.Error())
	}

	authenticator, err := authentication.New(conf.Server.CasAddr)
	if err != nil {
		log.Fatalf("create authenticator failed:%s", err.Error())
	}

	authorizer, err := authorization.New()
	if err != nil {
		log.Fatalf("create authorizer failed:%s", err.Error())
	}

	server, err := server.NewServer(authenticator.MiddlewareFunc())
	if err != nil {
		log.Fatalf("create server failed:%s", err.Error())
	}

	watcher := k8seventwatcher.New()
	if err := server.RegisterHandler(watcher); err != nil {
		log.Fatalf("register k8s event watcher failed:%s", err.Error())
	}

	shellExecutor := k8sshell.New()
	if err := server.RegisterHandler(shellExecutor); err != nil {
		log.Fatalf("register shell executor failed:%s", err.Error())
	}

	if err := server.RegisterHandler(clusteragent.GetAgent()); err != nil {
		log.Fatalf("register agent failed:%s", err.Error())
	}

	app, err := handler.NewApp(authenticator, authorizer, conf)
	if err != nil {
		log.Fatalf("create app failed %s", err.Error())
	}

	if err := server.RegisterHandler(authenticator); err != nil {
		log.Fatalf("register redirect handler failed:%s", err.Error())
	}

	if err := server.RegisterHandler(app); err != nil {
		log.Fatalf("register resource handler failed:%s", err.Error())
	}

	if err := server.Run(conf.Server.Addr); err != nil {
		log.Fatalf("server run failed:%s", err.Error())
	}
}

func runAsSlave(conf *config.SinglecloudConf) {
	db.RunAsSlave(conf)
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
