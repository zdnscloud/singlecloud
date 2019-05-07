package globaldns

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"

	"github.com/zdnscloud/cement/log"
)

const (
	EdgeNodeLabel = "node-role.kubernetes.io/edge"
)

var (
	EdgeNodeLabelSelector = &metav1.LabelSelector{MatchLabels: map[string]string{EdgeNodeLabel: "true"}}
)

type ClusterDNSSyncer struct {
	zoneName       string
	edgeNodeIPs    []string
	ingressDomains []string

	proxy *DnsProxy
}

func newClusterDNSSyncer(zoneName string, c cache.Cache, proxy *DnsProxy) (*ClusterDNSSyncer, error) {
	clusterDNSSyncer := &ClusterDNSSyncer{
		zoneName: zoneName,
		proxy:    proxy,
	}

	if err := clusterDNSSyncer.initClusterDNSSyncer(c); err != nil {
		return nil, err
	}

	ctrl := controller.New("globalDNSCache", c, scheme.Scheme)
	ctrl.Watch(&corev1.Node{})
	ctrl.Watch(&extv1beta1.Ingress{})
	stopCh := make(chan struct{})
	go ctrl.Start(stopCh, clusterDNSSyncer, predicate.NewIgnoreUnchangedUpdate())
	return clusterDNSSyncer, nil
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

	if err := c.proxy.AddAuthZone(c.zoneName); err != nil {
		return fmt.Errorf("add zone %s to globaldns failed: %s", c.zoneName, err.Error())
	}

	for _, node := range nodes.Items {
		c.OnNewNode(&node)
	}

	return nil
}

func (c *ClusterDNSSyncer) GetZoneName() string {
	return c.zoneName
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
		log.Warnf("new edge node %s address should not be empty", k8snode.Name)
		return
	}

	for _, ip := range c.edgeNodeIPs {
		if ip == nodeIP {
			return
		}
	}

	c.edgeNodeIPs = append(c.edgeNodeIPs, nodeIP)
	if err := c.proxy.AddAuthRRs(c.zoneName, c.ingressDomains, nodeIP); err != nil {
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
		log.Warnf("old edge node %s address should not be empty", k8snode.Name)
		return
	}

	for i, ip := range c.edgeNodeIPs {
		if ip == nodeIP {
			c.edgeNodeIPs = append(c.edgeNodeIPs[:i], c.edgeNodeIPs[i+1:]...)
			break
		}
	}

	if err := c.proxy.DeleteAuthRRs(c.zoneName, c.ingressDomains, nodeIP); err != nil {
		log.Errorf("delete all ingress rrsets with edge node %s failed: %v", k8snode.Name, err.Error())
	}
}

func (c *ClusterDNSSyncer) OnNewIngress(k8sing *extv1beta1.Ingress) {
	for _, rule := range k8sing.Spec.Rules {
		if strings.HasSuffix(rule.Host, c.zoneName) {
			if c.hasIngressDomain(rule.Host) {
				log.Warnf("duplicate ingress host %v with zone %v", rule.Host, c.zoneName)
				continue
			}

			c.ingressDomains = append(c.ingressDomains, rule.Host)
			log.Debugf("add new ingress host domain %v to zone %v", rule.Host, c.zoneName)
		} else {
			log.Warnf("add new ingress rrset failed: host domain %v not belong to zone %v", rule.Host, c.zoneName)
		}
	}

	if err := c.proxy.AddAuthRRs(c.zoneName, c.ingressDomains, c.edgeNodeIPs...); err != nil {
		log.Errorf("add new ingress rrsets failed: %v", err.Error())
	}
}

func (c *ClusterDNSSyncer) hasIngressDomain(domainName string) bool {
	for _, domain := range c.ingressDomains {
		if domain == domainName {
			return true
		}
	}
	return false
}

func (c *ClusterDNSSyncer) OnUpdateIngress(old, new *extv1beta1.Ingress) {
	oldDomains := diffK8sIngressRules(old.Spec.Rules, new.Spec.Rules)
	newDomains := diffK8sIngressRules(new.Spec.Rules, old.Spec.Rules)
	if len(oldDomains) == 0 && len(newDomains) == 0 {
		return
	}

	c.deleteIngressDomains(oldDomains)
	c.ingressDomains = append(c.ingressDomains, newDomains...)
	if err := c.proxy.UpdateAuthRRs(c.zoneName, oldDomains, newDomains, c.edgeNodeIPs...); err != nil {
		log.Errorf("update ingress %s rrsets failed: %v", old.Name, err.Error())
	}
}

func (c *ClusterDNSSyncer) deleteIngressDomains(domains []string) {
	for _, domain := range domains {
		for i, ingressDomain := range c.ingressDomains {
			if domain == ingressDomain {
				c.ingressDomains = append(c.ingressDomains[:i], c.ingressDomains[i+1:]...)
				break
			}
		}
	}
}

func (c *ClusterDNSSyncer) OnDeleteIngress(k8sing *extv1beta1.Ingress) {
	var oldDomains []string
	for _, rule := range k8sing.Spec.Rules {
		oldDomains = append(oldDomains, rule.Host)
		log.Debugf("delete ingress host domain %v from zone %v", rule.Host, c.zoneName)
	}

	c.deleteIngressDomains(oldDomains)
	if err := c.proxy.DeleteAuthRRs(c.zoneName, oldDomains, c.edgeNodeIPs...); err != nil {
		log.Errorf("delete ingress %s rrsets failed: %v", k8sing.Name, err.Error())
	}
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
