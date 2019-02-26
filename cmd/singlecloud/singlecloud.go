package main

import (
	"flag"

	"github.com/zdnscloud/singlecloud/config"
	"github.com/zdnscloud/singlecloud/logger"
	"github.com/zdnscloud/singlecloud/server"
)

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "c", "/etc/singlecloud.conf", "configure file path")
}

func main() {
	flag.Parse()
	conf, err := config.LoadConfig(configFile)
	if err != nil {
		panic("load configure file failed:" + err.Error())
	}

	if err := logger.InitLogger(conf); err != nil {
		panic("init logger failed:" + err.Error())
	}

	server, err := server.NewServer(conf)
	if err != nil {
		panic("create server failed:" + err.Error())
	}

	server.Run()
}
