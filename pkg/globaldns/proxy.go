package globaldns

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/vanguard/httpcmd"
	"github.com/zdnscloud/vanguard/resolver/auth"
	"github.com/zdnscloud/vanguard/server"
	view "github.com/zdnscloud/vanguard/viewselector"
)

const (
	cmdServiceName = "vanguard_cmd"
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

func (d *DnsProxy) AddAuthZone(zoneName *g53.Name) *httpcmd.Error {
	d.handleHttpCmd(&auth.DeleteAuthZone{View: view.DefaultView, Name: zoneName.String(false)})
	return d.handleHttpCmd(&auth.AddAuthZone{
		View: view.DefaultView,
		Name: zoneName.String(false)})
}

func (d *DnsProxy) AddAuthRRs(zoneName *g53.Name, domains []*g53.Name, ips ...string) *httpcmd.Error {
	if len(domains) == 0 || len(ips) == 0 {
		return nil
	}

	return d.handleHttpCmd(&auth.AddAuthRrs{Rrs: genAuthRRs(zoneName, domains, ips)})
}

func (d *DnsProxy) DeleteAuthRRs(zoneName *g53.Name, domains []*g53.Name, ips ...string) *httpcmd.Error {
	if len(domains) == 0 || len(ips) == 0 {
		return nil
	}

	return d.handleHttpCmd(&auth.DeleteAuthRrs{Rrs: genAuthRRs(zoneName, domains, ips)})
}

func (d *DnsProxy) UpdateAuthRRs(zoneName *g53.Name, oldDomains, newDomains []*g53.Name, ips ...string) *httpcmd.Error {
	if (len(oldDomains) == 0 && len(newDomains) == 0) || len(ips) == 0 {
		return nil
	}

	return d.handleHttpCmd(&auth.UpdateAuthRrs{
		OldRrs: genAuthRRs(zoneName, oldDomains, ips),
		NewRrs: genAuthRRs(zoneName, newDomains, ips)})
}

func genAuthRRs(zoneName *g53.Name, domains []*g53.Name, ips []string) auth.AuthRRs {
	var rrs auth.AuthRRs
	for _, domain := range domains {
		for _, ip := range ips {
			rrs = append(rrs, &auth.AuthRR{
				View:  view.DefaultView,
				Zone:  zoneName.String(false),
				Name:  domain.String(false),
				Ttl:   DefaultTtl,
				Type:  g53.RR_A.String(),
				Rdata: ip,
			})
		}
	}
	return rrs
}
