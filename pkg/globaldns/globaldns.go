package globaldns

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
)

const (
	KubeSystemNamespace = "kube-system"
	FullClusterState    = "full-cluster-state"
	EdgeNodeLabel       = "node-role.kubernetes.io/edge"
	True                = "true"
)

var (
	GetFullClusterStateOption = k8stypes.NamespacedName{KubeSystemNamespace, FullClusterState}
	EdgeNodeLabelSelector     = &metav1.LabelSelector{MatchLabels: map[string]string{EdgeNodeLabel: True}}
)

type GlobalDns struct {
	dnsCache *GlobalDnsCache
	lock     sync.RWMutex
	cache    cache.Cache
	stopCh   chan struct{}
}

func Init(c cache.Cache, httpCmdAddr string) error {
	ctrl := controller.New("globalDNSCache", c, scheme.Scheme)
	ctrl.Watch(&corev1.Node{})
	ctrl.Watch(&extv1beta1.Ingress{})

	stopCh := make(chan struct{})
	g := &GlobalDns{
		stopCh: stopCh,
		cache:  c,
	}
	if err := g.initDnsCache(httpCmdAddr); err != nil {
		return err
	}

	go ctrl.Start(stopCh, g, predicate.NewIgnoreUnchangedUpdate())
	return nil
}

func (g *GlobalDns) initDnsCache(httpCmdAddr string) error {
	nodes := &corev1.NodeList{}
	labels, err := metav1.LabelSelectorAsSelector(EdgeNodeLabelSelector)
	if err != nil {
		return fmt.Errorf("create edge node selector failed: %s", err.Error())
	}

	if err := g.cache.List(context.TODO(), &client.ListOptions{LabelSelector: labels}, nodes); err != nil {
		return fmt.Errorf("list all edge nodes failed: %s", err.Error())
	}

	k8sconfigmap := &corev1.ConfigMap{}
	if err := g.cache.Get(context.TODO(), GetFullClusterStateOption, k8sconfigmap); err != nil {
		return fmt.Errorf("get full-cluster-state configmap failed: %s", err.Error())
	}

	fullState := &FullState{}
	if err := json.Unmarshal([]byte(k8sconfigmap.Data[FullClusterState]), fullState); err != nil {
		return fmt.Errorf("unmarshal full-cluster-state configmap failed: %s", err.Error())
	}

	dnsCache, err := newGlobalDnsCache(fullState.DesiredState.ZKEConfig.Services.Kubelet.ClusterDomain, httpCmdAddr)
	if err != nil {
		return fmt.Errorf("init global dns cache failed: %s", err.Error())
	}

	for _, node := range nodes.Items {
		dnsCache.OnNewNode(&node)
	}
	g.dnsCache = dnsCache
	return nil
}

func (g *GlobalDns) OnCreate(e event.CreateEvent) (handler.Result, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Node:
		if obj.Labels[EdgeNodeLabel] == True {
			g.dnsCache.OnNewNode(obj)
		}
	case *extv1beta1.Ingress:
		g.dnsCache.OnNewIngress(obj)
	}

	return handler.Result{}, nil
}

func (g *GlobalDns) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	switch newObj := e.ObjectNew.(type) {
	case *extv1beta1.Ingress:
		g.dnsCache.OnUpdateIngress(e.ObjectOld.(*extv1beta1.Ingress), newObj)
	}

	return handler.Result{}, nil
}

func (g *GlobalDns) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Node:
		if obj.Labels[EdgeNodeLabel] == True {
			g.dnsCache.OnDeleteNode(obj)
		}
	case *extv1beta1.Ingress:
		g.dnsCache.OnDeleteIngress(obj)
	}

	return handler.Result{}, nil
}

func (g *GlobalDns) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}
