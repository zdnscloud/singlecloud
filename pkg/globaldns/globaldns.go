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
)

var (
	GetFullClusterStateOption = k8stypes.NamespacedName{KubeSystemNamespace, FullClusterState}
	EdgeNodeLabelSelector     = &metav1.LabelSelector{MatchLabels: map[string]string{EdgeNodeLabel: "true"}}
)

type GlobalDns struct {
	dnsSynchronizer *ClusterDNSSynchronizer
	lock            sync.RWMutex
	cache           cache.Cache
	stopCh          chan struct{}
}

func New(c cache.Cache, httpCmdAddr string) (*GlobalDns, error) {
	g := &GlobalDns{
		stopCh: make(chan struct{}),
		cache:  c,
	}
	if err := g.initDnsSynchronizer(httpCmdAddr); err != nil {
		return nil, err
	}

	return g, nil
}

func (g *GlobalDns) Run() {
	ctrl := controller.New("globalDNSCache", g.cache, scheme.Scheme)
	ctrl.Watch(&corev1.Node{})
	ctrl.Watch(&extv1beta1.Ingress{})
	ctrl.Start(g.stopCh, g, predicate.NewIgnoreUnchangedUpdate())
}

func (g *GlobalDns) initDnsSynchronizer(httpCmdAddr string) error {
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

	dnsSynchronizer, err := newClusterDNSSynchronizer(fullState.DesiredState.ZKEConfig.Services.Kubelet.ClusterDomain, httpCmdAddr)
	if err != nil {
		return fmt.Errorf("init global dns cache failed: %s", err.Error())
	}

	for _, node := range nodes.Items {
		dnsSynchronizer.OnNewNode(&node)
	}
	g.dnsSynchronizer = dnsSynchronizer
	return nil
}

func (g *GlobalDns) OnCreate(e event.CreateEvent) (handler.Result, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Node:
		if obj.Labels[EdgeNodeLabel] == "true" {
			g.dnsSynchronizer.OnNewNode(obj)
		}
	case *extv1beta1.Ingress:
		g.dnsSynchronizer.OnNewIngress(obj)
	}

	return handler.Result{}, nil
}

func (g *GlobalDns) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	switch newObj := e.ObjectNew.(type) {
	case *extv1beta1.Ingress:
		g.dnsSynchronizer.OnUpdateIngress(e.ObjectOld.(*extv1beta1.Ingress), newObj)
	}

	return handler.Result{}, nil
}

func (g *GlobalDns) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Node:
		if obj.Labels[EdgeNodeLabel] == "true" {
			g.dnsSynchronizer.OnDeleteNode(obj)
		}
	case *extv1beta1.Ingress:
		g.dnsSynchronizer.OnDeleteIngress(obj)
	}

	return handler.Result{}, nil
}

func (g *GlobalDns) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}
