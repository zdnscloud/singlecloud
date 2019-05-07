package globaldns

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zdnscloud/vanguard/httpcmd"
	"github.com/zdnscloud/vanguard/resolver/auth"
	"github.com/zdnscloud/vanguard/server"
)

const (
	cmdServiceName = "vanguard_cmd"
	DefaultView    = "default"
	RRTypeA        = "A"
	DefaultTtl     = "3600"
)

var supportedCommands = []httpcmd.Command{
	&server.Ping{},
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

	dnsProxy := &DnsProxy{proxy: proxy}
	if err := dnsProxy.handleHttpCmd(&server.Ping{}); err != nil {
		return nil, fmt.Errorf("new globaldns proxy failed: %v", err.Error())
	}

	return dnsProxy, nil
}

func (d *DnsProxy) handleHttpCmd(command httpcmd.Command) *httpcmd.Error {
	task := httpcmd.NewTask()
	task.AddCmd(command)

	return d.proxy.HandleTask(task, nil)
}

func (d *DnsProxy) AddAuthZone(zoneName string) *httpcmd.Error {
	d.handleHttpCmd(&auth.DeleteAuthZone{View: DefaultView, Name: zoneName})
	return d.handleHttpCmd(&auth.AddAuthZone{
		View: DefaultView,
		Name: zoneName})
}

func (d *DnsProxy) AddAuthRRs(zoneName string, domains []string, ips ...string) *httpcmd.Error {
	if len(domains) == 0 || len(ips) == 0 {
		return nil
	}

	return d.handleHttpCmd(&auth.AddAuthRrs{Rrs: genAuthRRs(zoneName, domains, ips)})
}

func (d *DnsProxy) DeleteAuthRRs(zoneName string, domains []string, ips ...string) *httpcmd.Error {
	if len(domains) == 0 || len(ips) == 0 {
		return nil
	}

	return d.handleHttpCmd(&auth.DeleteAuthRrs{Rrs: genAuthRRs(zoneName, domains, ips)})
}

func (d *DnsProxy) UpdateAuthRRs(zoneName string, oldDomains, newDomains []string, ips ...string) *httpcmd.Error {
	if (len(oldDomains) == 0 && len(newDomains) == 0) || len(ips) == 0 {
		return nil
	}

	return d.handleHttpCmd(&auth.UpdateAuthRrs{
		OldRrs: genAuthRRs(zoneName, oldDomains, ips),
		NewRrs: genAuthRRs(zoneName, newDomains, ips)})
}

func genAuthRRs(zoneName string, domains, ips []string) auth.AuthRRs {
	var rrs auth.AuthRRs
	for _, domain := range domains {
		for _, ip := range ips {
			rrs = append(rrs, &auth.AuthRR{
				View:  DefaultView,
				Zone:  zoneName,
				Name:  domain,
				Ttl:   DefaultTtl,
				Type:  RRTypeA,
				Rdata: ip,
			})
		}
	}
	return rrs
}
