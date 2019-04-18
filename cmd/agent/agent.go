package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/zdnscloud/goproxy"
)

var (
	addr    string
	cluster string
	id      string
)

func main() {
	flag.StringVar(&addr, "addr", "", "single cloud server addr")
	flag.StringVar(&cluster, "cluster", "local", "cluster this agent belongs to")
	flag.StringVar(&id, "id", "", "agent id with identify this agent")
	flag.Parse()

	url := fmt.Sprintf("ws://%s/apis/agent.zcloud.cn/v1/clusters/%s/register/%s", addr, cluster, id)
	err := goproxy.RegisterAgent(url, func(string, string) bool { return true }, nil)
	if err != nil {
		log.Printf("agent exist with err: %s", err.Error())
	}
}
