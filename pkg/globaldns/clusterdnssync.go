package globaldns

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/vanguard/resolver/auth"
)

const (
	DefaultView = "default"
	RRTypeA     = "A"
	DefaultTtl  = "3600"
)

type ClusterDNSSynchronizer struct {
	zoneName       string
	edgeNodeIPs    []string
	ingressDomains map[string]struct{}

	proxy *DnsProxy
}

func newClusterDNSSynchronizer(zoneName, httpCmdAddr string) (*ClusterDNSSynchronizer, error) {
	if zoneName == "" {
		return nil, fmt.Errorf("cluster domain should not be empty")
	}

	proxy, err := newDnsProxy(httpCmdAddr)
	if err != nil {
		return nil, fmt.Errorf("connect globaldns failed: %s", err.Error())
	}

	proxy.HandleHttpCmd(&auth.DeleteAuthZone{View: DefaultView, Name: zoneName})
	if err := proxy.HandleHttpCmd(&auth.AddAuthZone{
		View: DefaultView,
		Name: zoneName}); err != nil {
		return nil, fmt.Errorf("add zone %s to globaldns failed: %s", zoneName, err.Error())
	}

	return &ClusterDNSSynchronizer{
		zoneName:       zoneName,
		ingressDomains: make(map[string]struct{}),
		proxy:          proxy,
	}, nil
}

func (c *ClusterDNSSynchronizer) OnNewNode(k8snode *corev1.Node) {
	nodeIP := getK8sNodeIP(k8snode)
	for _, ip := range c.edgeNodeIPs {
		if ip == nodeIP {
			return
		}
	}

	c.edgeNodeIPs = append(c.edgeNodeIPs, nodeIP)
	if len(c.ingressDomains) == 0 {
		return
	}

	var newAuthRRs auth.AuthRRs
	for domain, _ := range c.ingressDomains {
		newAuthRRs = append(newAuthRRs, c.genAuthRRs(domain, []string{nodeIP})...)
	}

	if err := c.proxy.HandleHttpCmd(&auth.AddAuthRrs{
		Rrs: newAuthRRs}); err != nil {
		log.Errorf("add ingress rrsets when add new edge node %s failed: %v", k8snode.Name, err.Error())
	}
}

func getK8sNodeIP(k8snode *corev1.Node) string {
	var ip string
	for _, addr := range k8snode.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP || addr.Type == corev1.NodeExternalIP {
			if ip == "" {
				ip = addr.Address
			}
		}
	}
	return ip
}

func (c *ClusterDNSSynchronizer) OnDeleteNode(k8snode *corev1.Node) {
	nodeIP := getK8sNodeIP(k8snode)
	for i, ip := range c.edgeNodeIPs {
		if ip == nodeIP {
			c.edgeNodeIPs = append(c.edgeNodeIPs[:i], c.edgeNodeIPs[i+1:]...)
			break
		}
	}
	if len(c.ingressDomains) == 0 {
		return
	}

	var oldAuthRRs auth.AuthRRs
	for domain, _ := range c.ingressDomains {
		oldAuthRRs = append(oldAuthRRs, c.genAuthRRs(domain, []string{nodeIP})...)
	}

	if err := c.proxy.HandleHttpCmd(&auth.DeleteAuthRrs{
		Rrs: oldAuthRRs}); err != nil {
		log.Errorf("delete all ingress rrsets with edge node %s failed: %v", k8snode.Name, err.Error())
	}
}

func (c *ClusterDNSSynchronizer) OnNewIngress(k8sing *extv1beta1.Ingress) {
	var newAuthRRs auth.AuthRRs
	for _, rule := range k8sing.Spec.Rules {
		if strings.HasSuffix(rule.Host, c.zoneName) {
			newAuthRRs = append(newAuthRRs, c.genAuthRRs(rule.Host, c.edgeNodeIPs)...)
			c.ingressDomains[rule.Host] = struct{}{}
		}
	}

	if len(newAuthRRs) != 0 {
		if err := c.proxy.HandleHttpCmd(&auth.AddAuthRrs{
			Rrs: newAuthRRs}); err != nil {
			log.Errorf("add new ingress rrsets failed: %v", err.Error())
		}
	}
}

func (c *ClusterDNSSynchronizer) OnUpdateIngress(old, new *extv1beta1.Ingress) {
	oldDomains := diffK8sIngressRules(old.Spec.Rules, new.Spec.Rules)
	newDomains := diffK8sIngressRules(new.Spec.Rules, old.Spec.Rules)
	if len(oldDomains) == 0 && len(newDomains) == 0 {
		return
	}

	var oldAuthRRs, newAuthRRs auth.AuthRRs
	for _, oldDomain := range oldDomains {
		oldAuthRRs = append(oldAuthRRs, c.genAuthRRs(oldDomain, c.edgeNodeIPs)...)
		delete(c.ingressDomains, oldDomain)
	}

	for _, newDomain := range newDomains {
		newAuthRRs = append(newAuthRRs, c.genAuthRRs(newDomain, c.edgeNodeIPs)...)
		c.ingressDomains[newDomain] = struct{}{}
	}

	if err := c.proxy.HandleHttpCmd(&auth.UpdateAuthRrs{
		OldRrs: oldAuthRRs,
		NewRrs: newAuthRRs}); err != nil {
		log.Errorf("update ingress %s rrsets failed: %v", old.Name, err.Error())
	}
}

func (c *ClusterDNSSynchronizer) OnDeleteIngress(k8sing *extv1beta1.Ingress) {
	var oldAuthRRs auth.AuthRRs
	for _, rule := range k8sing.Spec.Rules {
		oldAuthRRs = append(oldAuthRRs, c.genAuthRRs(rule.Host, c.edgeNodeIPs)...)
		delete(c.ingressDomains, rule.Host)
	}

	if err := c.proxy.HandleHttpCmd(&auth.DeleteAuthRrs{
		Rrs: oldAuthRRs}); err != nil {
		log.Errorf("delete ingress %s rrsets failed: %v", k8sing.Name, err.Error())
	}
}

func (c *ClusterDNSSynchronizer) genAuthRRs(domain string, ips []string) auth.AuthRRs {
	var rrs auth.AuthRRs
	for _, ip := range ips {
		rrs = append(rrs, &auth.AuthRR{
			View:  DefaultView,
			Zone:  c.zoneName,
			Name:  domain,
			Ttl:   DefaultTtl,
			Type:  RRTypeA,
			Rdata: ip,
		})
	}

	return rrs
}

func diffK8sIngressRules(rules1, rules2 []extv1beta1.IngressRule) []string {
	var diffHosts []string
	for _, r1 := range rules1 {
		found := false
		for _, r2 := range rules2 {
			if r1.Host == r2.Host {
				found = true
				break
			}
		}

		if found == false {
			diffHosts = append(diffHosts, r1.Host)
		}
	}

	return diffHosts
}
