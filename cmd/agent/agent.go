package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/goproxy"
)

var (
	addr     string
	cluster  string
	agentKey string
)

const (
	RetryInterval = 10 * time.Second
)

func main() {
	flag.StringVar(&addr, "server", "127.0.0.1:80", "single cloud server addr")
	flag.StringVar(&cluster, "cluster", "local", "cluster this agent belongs to")
	flag.Parse()

	log.InitLogger(log.Debug)

	url := fmt.Sprintf("wss://%s/apis/agent.zcloud.cn/v1/register/%s", addr, cluster)
	for {
		err := goproxy.RegisterAgent(url, func(string, string) bool { return true }, nil)
		if err != nil {
			log.Warnf("agent %s connect to single cloud %s failed:%s, start to retry", cluster, addr, err.Error())
		}
		<-time.After(RetryInterval)
	}
}
