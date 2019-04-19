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
	flag.StringVar(&agentKey, "agentKey", "", "id which identify this agent")
	flag.Parse()

	log.InitLogger(log.Debug)

	url := fmt.Sprintf("ws://%s/apis/agent.zcloud.cn/v1/clusters/%s/register/%s", addr, cluster, agentKey)
	for {
		err := goproxy.RegisterAgent(url, func(string, string) bool { return true }, nil)
		if err != nil {
			log.Warnf("agent %s connect to cluster %s(%s) failed:%s, start to retry", agentKey, cluster, addr, err.Error())
		}
		<-time.After(RetryInterval)
	}
}
