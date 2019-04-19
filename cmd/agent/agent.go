package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/zdnscloud/goproxy"
	"github.com/zdnscloud/singlecloud/pkg/logger"
)

var (
	addr    string
	cluster string
	id      string
)

const (
	RetryInterval = 10 * time.Second
)

func main() {
	flag.StringVar(&addr, "addr", "127.0.0.1:80", "single cloud server addr")
	flag.StringVar(&cluster, "cluster", "local", "cluster this agent belongs to")
	flag.StringVar(&id, "id", "", "agent id with identify this agent")
	flag.Parse()

	logger.InitLogger()

	url := fmt.Sprintf("ws://%s/apis/agent.zcloud.cn/v1/clusters/%s/register/%s", addr, cluster, id)
	for {
		err := goproxy.RegisterAgent(url, func(string, string) bool { return true }, nil)
		if err != nil {
			logger.Warn("agent %s connect to cluster %s(%s) failed:%s, start to retry", id, cluster, addr, err.Error())
		}
		<-time.After(RetryInterval)
	}
}
