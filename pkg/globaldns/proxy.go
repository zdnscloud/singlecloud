package globaldns

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zdnscloud/vanguard/httpcmd"
	"github.com/zdnscloud/vanguard/resolver/auth"
)

const (
	cmdServiceName = "vanguard_cmd"
)

var supportedCommands = []httpcmd.Command{
	&auth.AddAuthZone{},
	&auth.AddAuthRrs{},
	&auth.DeleteAuthRrs{},
	&auth.UpdateAuthRrs{},
}

type DnsProxy struct {
	proxy *httpcmd.HttpCmdProxy
}

func newDnsProxy(addr string) (*DnsProxy, error) {
	ipAndPort := strings.Split(addr, ":")
	if len(ipAndPort) != 2 {
		return nil, fmt.Errorf("globaldns httpcmd addr %s is invalid", addr)
	}

	port, err := strconv.Atoi(ipAndPort[1])
	if err != nil {
		return nil, fmt.Errorf("globaldns httpcmd port %v is invalid: %v", ipAndPort[1], err.Error())
	}

	proxy, err := httpcmd.GetProxy(&httpcmd.EndPoint{
		Name: cmdServiceName,
		IP:   ipAndPort[0],
		Port: port,
	}, supportedCommands)
	if err != nil {
		return nil, fmt.Errorf("new globaldns proxy failed: %v", err.Error())
	}

	return &DnsProxy{
		proxy: proxy,
	}, nil
}

func (d *DnsProxy) HandleHttpCmd(command httpcmd.Command) *httpcmd.Error {
	task := httpcmd.NewTask()
	task.AddCmd(command)

	return d.proxy.HandleTask(task, nil)
}
