package globaldns

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/vanguard/resolver/auth"
)

const (
	DefaultView         = "default"
	RRTypeA             = "A"
	DefaultTtl          = "3600"
	KubeSystemNamespace = "kube-system"
	FullClusterState    = "full-cluster-state"
	EdgeNodeLabel       = "node-role.kubernetes.io/edge"
)

var (
	GetFullClusterStateOption = k8stypes.NamespacedName{KubeSystemNamespace, FullClusterState}
	EdgeNodeLabelSelector     = &metav1.LabelSelector{MatchLabels: map[string]string{EdgeNodeLabel: "true"}}
)

type ClusterDNSSyncer struct {
	zoneName       string
	edgeNodeIPs    []string
	ingressDomains map[string]struct{}

	proxy *DnsProxy
}

func New(c cache.Cache, httpCmdAddr string) error {
	proxy, err := newDnsProxy(httpCmdAddr)
	if err != nil {
		return fmt.Errorf("connect globaldns failed: %s", err.Error())
	}

	clusterDNSSyncer := &ClusterDNSSyncer{
		ingressDomains: make(map[string]struct{}),
		proxy:          proxy,
	}

	if err := clusterDNSSyncer.initClusterDNSSyncer(c); err != nil {
		return err
	}

	ctrl := controller.New("globalDNSCache", c, scheme.Scheme)
	ctrl.Watch(&corev1.Node{})
	ctrl.Watch(&extv1beta1.Ingress{})
	stopCh := make(chan struct{})
	go ctrl.Start(stopCh, clusterDNSSyncer, predicate.NewIgnoreUnchangedUpdate())
	return nil
}

func (c *ClusterDNSSyncer) initClusterDNSSyncer(cache cache.Cache) error {
	nodes := &corev1.NodeList{}
	labels, err := metav1.LabelSelectorAsSelector(EdgeNodeLabelSelector)
	if err != nil {
		return fmt.Errorf("create edge node selector failed: %s", err.Error())
	}

	if err := cache.List(context.TODO(), &client.ListOptions{LabelSelector: labels}, nodes); err != nil {
		return fmt.Errorf("list all edge nodes failed: %s", err.Error())
	}

	k8sconfigmap := &corev1.ConfigMap{}
	if err := cache.Get(context.TODO(), GetFullClusterStateOption, k8sconfigmap); err != nil {
		return fmt.Errorf("get full-cluster-state configmap failed: %s", err.Error())
	}

	fullState := &FullState{}
	if err := json.Unmarshal([]byte(k8sconfigmap.Data[FullClusterState]), fullState); err != nil {
		return fmt.Errorf("unmarshal full-cluster-state configmap failed: %s", err.Error())
	}

	zoneName := fullState.DesiredState.ZKEConfig.Services.Kubelet.ClusterDomain
	if zoneName == "" {
		return fmt.Errorf("cluster domain should not be empty")
	}

	c.proxy.HandleHttpCmd(&auth.DeleteAuthZone{View: DefaultView, Name: zoneName})
	if err := c.proxy.HandleHttpCmd(&auth.AddAuthZone{
		View: DefaultView,
		Name: zoneName}); err != nil {
		return fmt.Errorf("add zone %s to globaldns failed: %s", zoneName, err.Error())
	}

	for _, node := range nodes.Items {
		c.OnNewNode(&node)
	}

	c.zoneName = zoneName
	return nil
}

func (c *ClusterDNSSyncer) OnCreate(e event.CreateEvent) (handler.Result, error) {
	switch obj := e.Object.(type) {
	case *corev1.Node:
		if obj.Labels[EdgeNodeLabel] == "true" {
			c.OnNewNode(obj)
		}
	case *extv1beta1.Ingress:
		c.OnNewIngress(obj)
	}

	return handler.Result{}, nil
}

func (c *ClusterDNSSyncer) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	switch newObj := e.ObjectNew.(type) {
	case *extv1beta1.Ingress:
		c.OnUpdateIngress(e.ObjectOld.(*extv1beta1.Ingress), newObj)
	}

	return handler.Result{}, nil
}

func (c *ClusterDNSSyncer) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	switch obj := e.Object.(type) {
	case *corev1.Node:
		if obj.Labels[EdgeNodeLabel] == "true" {
			c.OnDeleteNode(obj)
		}
	case *extv1beta1.Ingress:
		c.OnDeleteIngress(obj)
	}

	return handler.Result{}, nil
}

func (c *ClusterDNSSyncer) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (c *ClusterDNSSyncer) OnNewNode(k8snode *corev1.Node) {
	nodeIP := getK8sNodeIP(k8snode)
	if nodeIP == "" {
		log.Warnf("new edge node %s address should not be zero", k8snode.Name)
		return
	}

	for _, ip := range c.edgeNodeIPs {
		if ip == nodeIP {
			log.Warnf("new edge node %s address %s has exist", k8snode.Name, nodeIP)
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
	for _, addr := range k8snode.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP || addr.Type == corev1.NodeExternalIP {
			if addr.Address != "" {
				return addr.Address
			}
		}
	}
	return ""
}

func (c *ClusterDNSSyncer) OnDeleteNode(k8snode *corev1.Node) {
	nodeIP := getK8sNodeIP(k8snode)
	if nodeIP == "" {
		log.Warnf("old edge node %s address should not be zero", k8snode.Name)
		return
	}

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

func (c *ClusterDNSSyncer) OnNewIngress(k8sing *extv1beta1.Ingress) {
	var newAuthRRs auth.AuthRRs
	for _, rule := range k8sing.Spec.Rules {
		if strings.HasSuffix(rule.Host, c.zoneName) {
			newAuthRRs = append(newAuthRRs, c.genAuthRRs(rule.Host, c.edgeNodeIPs)...)
			c.ingressDomains[rule.Host] = struct{}{}
			log.Debugf("add new ingress host domain %v to zone %v", rule.Host, c.zoneName)
		} else {
			log.Warnf("add new ingress rrset failed: host domain %v not belong to zone %v", rule.Host, c.zoneName)
		}
	}

	if len(newAuthRRs) != 0 {
		if err := c.proxy.HandleHttpCmd(&auth.AddAuthRrs{
			Rrs: newAuthRRs}); err != nil {
			log.Errorf("add new ingress rrsets failed: %v", err.Error())
		}
	}
}

func (c *ClusterDNSSyncer) OnUpdateIngress(old, new *extv1beta1.Ingress) {
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

func (c *ClusterDNSSyncer) OnDeleteIngress(k8sing *extv1beta1.Ingress) {
	var oldAuthRRs auth.AuthRRs
	for _, rule := range k8sing.Spec.Rules {
		oldAuthRRs = append(oldAuthRRs, c.genAuthRRs(rule.Host, c.edgeNodeIPs)...)
		delete(c.ingressDomains, rule.Host)
		log.Debugf("delete ingress host domain %v from zone %v", rule.Host, c.zoneName)
	}

	if err := c.proxy.HandleHttpCmd(&auth.DeleteAuthRrs{
		Rrs: oldAuthRRs}); err != nil {
		log.Errorf("delete ingress %s rrsets failed: %v", k8sing.Name, err.Error())
	}
}

func (c *ClusterDNSSyncer) genAuthRRs(domain string, ips []string) auth.AuthRRs {
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
