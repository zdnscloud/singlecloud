package globaldns

import (
	"context"
	"fmt"

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
	"github.com/zdnscloud/g53"
)

const (
	EdgeNodeLabel = "node-role.kubernetes.io/edge"
)

var (
	EdgeNodeLabelSelector = &metav1.LabelSelector{MatchLabels: map[string]string{EdgeNodeLabel: "true"}}
	ErrNotInZone          = fmt.Errorf("domain not belongs to zone")
)

type ClusterDNSSyncer struct {
	zoneName       *g53.Name
	edgeNodeIPs    []string
	ingressDomains []*g53.Name
	proxy          *DnsProxy
	stopCh         chan struct{}
}

func newClusterDNSSyncer(zoneName *g53.Name, c cache.Cache, proxy *DnsProxy) (*ClusterDNSSyncer, error) {
	stopCh := make(chan struct{})
	clusterDNSSyncer := &ClusterDNSSyncer{
		zoneName: zoneName,
		proxy:    proxy,
		stopCh:   stopCh,
	}

	if err := clusterDNSSyncer.initClusterDNSSyncer(c); err != nil {
		return nil, err
	}

	ctrl := controller.New("globalDNSCache", c, scheme.Scheme)
	ctrl.Watch(&corev1.Node{})
	ctrl.Watch(&extv1beta1.Ingress{})
	go ctrl.Start(stopCh, clusterDNSSyncer, predicate.NewIgnoreUnchangedUpdate())
	return clusterDNSSyncer, nil
}

func (c *ClusterDNSSyncer) Stop() {
	if err := c.proxy.DeleteAuthZone(c.zoneName); err != nil {
		log.Warnf("delete zone %v from globaldns failed: %v", c.zoneName.String(false), err.Error())
	} else {
		log.Debugf("delete zone %v from globaldns", c.zoneName.String(false))
	}
	close(c.stopCh)
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
		return fmt.Errorf("add zone %s to globaldns failed: %s", c.zoneName.String(false), err.Error())
	} else {
		log.Debugf("add zone %v to globaldns", c.zoneName.String(false))
	}

	for _, node := range nodes.Items {
		c.OnNewNode(&node)
	}

	return nil
}

func (c *ClusterDNSSyncer) GetZoneName() *g53.Name {
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
	case *corev1.Node:
		if newObj.Labels[EdgeNodeLabel] == "true" {
			c.OnUpdateNode(e.ObjectOld.(*corev1.Node), newObj)
		}
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
	if isNodeReady(k8snode) == false {
		return
	}

	c.addNode(k8snode)
	return
}

func (c *ClusterDNSSyncer) addNode(k8snode *corev1.Node) {
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

	log.Debugf("add new edge node %v with ip %v", k8snode.Name, nodeIP)
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

func (c *ClusterDNSSyncer) OnUpdateNode(oldK8snode, newK8snode *corev1.Node) {
	if newIsReady := isNodeReady(newK8snode); newIsReady != isNodeReady(oldK8snode) {
		if newIsReady {
			c.addNode(newK8snode)
		} else {
			c.deleteNode(newK8snode)
		}
	}
}

func isNodeReady(k8snode *corev1.Node) bool {
	for _, cond := range k8snode.Status.Conditions {
		if cond.Type == corev1.NodeReady &&
			cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (c *ClusterDNSSyncer) OnDeleteNode(k8snode *corev1.Node) {
	if isNodeReady(k8snode) == false {
		return
	}

	c.deleteNode(k8snode)
	return
}

func (c *ClusterDNSSyncer) deleteNode(k8snode *corev1.Node) {
	nodeIP := getK8sNodeIP(k8snode)
	if nodeIP == "" {
		log.Warnf("delete edge node %s address should not be empty", k8snode.Name)
		return
	}

	isExist := false
	for i, ip := range c.edgeNodeIPs {
		if ip == nodeIP {
			c.edgeNodeIPs = append(c.edgeNodeIPs[:i], c.edgeNodeIPs[i+1:]...)
			isExist = true
			break
		}
	}

	if isExist == false {
		log.Warnf("delete edge node %s with ip %v is unknown", k8snode.Name, nodeIP)
		return
	}

	log.Debugf("delete edge node %v with ip %v", k8snode.Name, nodeIP)
	if err := c.proxy.DeleteAuthRRs(c.zoneName, c.ingressDomains, nodeIP); err != nil {
		log.Errorf("delete all ingress rrsets with edge node %s failed: %v", k8snode.Name, err.Error())
	}
}

func (c *ClusterDNSSyncer) OnNewIngress(k8sing *extv1beta1.Ingress) {
	var newDomains []*g53.Name
	for _, rule := range k8sing.Spec.Rules {
		hostDomain, exist, err := c.k8sIngressHostToSCDomain(rule.Host)
		if err == nil {
			if exist {
				log.Warnf("duplicate ingress host %v with zone %v", rule.Host, c.zoneName.String(false))
				continue
			}
			newDomains = append(newDomains, hostDomain)
			log.Debugf("add new ingress host domain %v to zone %v", rule.Host, c.zoneName.String(false))
		} else if err == ErrNotInZone {
			log.Warnf("add new ingress rrset failed: host domain %v not belong to zone %v", rule.Host, c.zoneName.String(false))
			continue
		} else {
			log.Errorf("add new ingress with host %s failed: %v", rule.Host, err.Error())
			return
		}
	}

	c.ingressDomains = append(c.ingressDomains, newDomains...)
	if err := c.proxy.AddAuthRRs(c.zoneName, newDomains, c.edgeNodeIPs...); err != nil {
		log.Errorf("add new ingress rrsets failed: %v", err.Error())
	}
}

func (c *ClusterDNSSyncer) k8sIngressHostToSCDomain(host string) (*g53.Name, bool, error) {
	hostDomain, err := g53.NameFromString(host)
	if err != nil {
		return nil, false, err
	}

	if hostDomain.IsSubDomain(c.zoneName) == false {
		return nil, false, ErrNotInZone
	}

	return hostDomain, c.hasIngressDomain(hostDomain), nil
}

func (c *ClusterDNSSyncer) hasIngressDomain(domainName *g53.Name) bool {
	for _, domain := range c.ingressDomains {
		if domain.Equals(domainName) {
			return true
		}
	}
	return false
}

func (c *ClusterDNSSyncer) OnUpdateIngress(old, new *extv1beta1.Ingress) {
	oldDomains, err := c.diffK8sIngressRules(old.Spec.Rules, new.Spec.Rules, true)
	if err != nil {
		return
	}

	newDomains, err := c.diffK8sIngressRules(new.Spec.Rules, old.Spec.Rules, false)
	if err != nil {
		return
	}

	if len(oldDomains) == 0 && len(newDomains) == 0 {
		return
	}

	c.deleteIngressDomains(oldDomains)
	c.ingressDomains = append(c.ingressDomains, newDomains...)
	if err := c.proxy.UpdateAuthRRs(c.zoneName, oldDomains, newDomains, c.edgeNodeIPs...); err != nil {
		log.Errorf("update ingress %s rrsets failed: %v", old.Name, err.Error())
	}
}

func (c *ClusterDNSSyncer) deleteIngressDomains(domains []*g53.Name) {
	for _, domain := range domains {
		for i, ingressDomain := range c.ingressDomains {
			if domain.Equals(ingressDomain) {
				c.ingressDomains = append(c.ingressDomains[:i], c.ingressDomains[i+1:]...)
				break
			}
		}
	}
}

func (c *ClusterDNSSyncer) OnDeleteIngress(k8sing *extv1beta1.Ingress) {
	var oldDomains []*g53.Name
	for _, rule := range k8sing.Spec.Rules {
		hostDomain, exist, err := c.k8sIngressHostToSCDomain(rule.Host)
		if err == nil {
			if exist == false {
				log.Warnf("no found ingress host %v with zone %v", rule.Host, c.zoneName.String(false))
				continue
			}
			log.Debugf("delete ingress host domain %v from zone %v", rule.Host, c.zoneName.String(false))
			oldDomains = append(oldDomains, hostDomain)
		} else if err == ErrNotInZone {
			log.Warnf("delete ingress rrset failed: host domain %v not belong to zone %v", rule.Host, c.zoneName.String(false))
			continue
		} else {
			log.Errorf("delete ingress with host %s failed: %v", rule.Host, err.Error())
			return
		}
	}

	c.deleteIngressDomains(oldDomains)
	if err := c.proxy.DeleteAuthRRs(c.zoneName, oldDomains, c.edgeNodeIPs...); err != nil {
		log.Errorf("delete ingress %s rrsets failed: %v", k8sing.Name, err.Error())
	}
}

func (c *ClusterDNSSyncer) diffK8sIngressRules(rules1, rules2 []extv1beta1.IngressRule, needExist bool) ([]*g53.Name, error) {
	var diffHosts []*g53.Name
	for _, r1 := range rules1 {
		found := false
		for _, r2 := range rules2 {
			if r1.Host == r2.Host {
				found = true
				break
			}
		}

		if found == false {
			hostDomain, err := g53.NameFromString(r1.Host)
			if err != nil {
				log.Errorf("parse ingress host %s failed: %v", r1.Host, err.Error())
				return nil, err
			}

			if hostDomain.IsSubDomain(c.zoneName) == false {
				continue
			}

			if needExist {
				if c.hasIngressDomain(hostDomain) == false {
					continue
				}
			} else {
				if c.hasIngressDomain(hostDomain) {
					continue
				}
			}

			diffHosts = append(diffHosts, hostDomain)
		}
	}

	return diffHosts, nil
}
