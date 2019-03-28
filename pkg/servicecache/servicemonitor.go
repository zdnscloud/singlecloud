package resourcerepo

import (
	"context"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/singlecloud/pkg/logger"
)

type ServiceMonitor struct {
	services map[string]*Service
	lock     sync.Mutex

	cache cache.Cache
}

func newServiceMonitor(cache cache.Cache) *ServiceMonitor {
	return &ServiceMonitor{
		cache:    cache,
		services: make(map[string]*Service),
	}
}

func (s *ServiceMonitor) GetServices() []*Service {
	s.lock.Lock()
	defer s.lock.Unlock()
	svcs := make([]*Service, 0, len(s.services))
	for _, svc := range s.services {
		svcs = append(svcs, svc)
	}
	return svcs
}

func (s *ServiceMonitor) OnNewService(k8ssvc *corev1.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	svc := &Service{
		Name: k8ssvc.Name,
	}

	ls := metav1.LabelSelector{
		MatchLabels: k8ssvc.Spec.Selector,
	}
	k8spods := corev1.PodList{}
	opts := &client.ListOptions{Namespace: k8ssvc.Namespace}
	labels, _ := metav1.LabelSelectorAsSelector(&ls)
	opts.LabelSelector = labels
	err := s.cache.List(context.TODO(), opts, &k8spods)
	if err != nil {
		logger.Warn("get pod list failed:%s", err.Error())
		return
	}

	workerLoads := make(map[string]*Workload)
	for _, k8spod := range k8spods.Items {
		pod := &Pod{
			Name:    k8spod.Name,
			IsReady: k8spod.Status.Phase == corev1.PodRunning,
		}

		if len(k8spod.OwnerReferences) == 1 {
			name, kind, succeed := s.getPodOwner(k8spod.Namespace, k8spod.OwnerReferences[0])
			if succeed == false {
				continue
			}

			wl, ok := workerLoads[name]
			if ok == false {
				wl = &Workload{
					Name: name,
					Kind: kind,
				}
				svc.Workloads = append(svc.Workloads, wl)
				workerLoads[name] = wl
			}
			wl.Pods = append(wl.Pods, pod)
		}
	}
	s.services[svc.Name] = svc
}

func (s *ServiceMonitor) getPodOwner(namespace string, owner metav1.OwnerReference) (string, string, bool) {
	if owner.Kind != "ReplicaSet" {
		return owner.Name, owner.Kind, true
	}

	var k8srs appsv1.ReplicaSet
	err := s.cache.Get(context.TODO(), k8stypes.NamespacedName{namespace, owner.Name}, &k8srs)
	if err != nil {
		logger.Warn("get replicaset failed:%s", err.Error())
		return "", "", false
	}

	if len(k8srs.OwnerReferences) != 1 {
		logger.Warn("replicaset OwnerReferences is strange:%v", k8srs.OwnerReferences)
		return "", "", false
	}

	owner = k8srs.OwnerReferences[0]
	if owner.Kind != "Deployment" {
		logger.Warn("replicaset parent is not deployment but %v", owner.Kind)
		return "", "", false
	}
	return owner.Name, owner.Kind, true
}

func (s *ServiceMonitor) OnDeleteService(svc *corev1.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.services, svc.Name)
}

func (s *ServiceMonitor) OnUpdateService(old, new *corev1.Service) {
	if isMapEqual(old.Spec.Selector, new.Spec.Selector) {
		return
	}
	s.OnNewService(new)
}

func (s *ServiceMonitor) OnUpdateEndpoints(old, new *corev1.Endpoints) {
	if len(old.Subsets) == 0 && len(new.Subsets) == 0 {
		return
	}

	s.lock.Lock()
	hasPodChange := s.hasPodNameChange(new)
	s.lock.Unlock()

	if hasPodChange {
		var k8ssvc corev1.Service
		err := s.cache.Get(context.TODO(), k8stypes.NamespacedName{new.Namespace, new.Name}, &k8ssvc)
		if err != nil {
			logger.Warn("get service %s failed:%s", new.Name, err.Error())
			return
		}
		s.OnNewService(&k8ssvc)
	}
}

func (s *ServiceMonitor) hasPodNameChange(eps *corev1.Endpoints) bool {
	svc, ok := s.services[eps.Name]
	if ok == false {
		logger.Warn("endpoints %s has no related service", eps.Name)
		return false
	}

	pods := make(map[string]*Pod)
	for _, wl := range svc.Workloads {
		for _, p := range wl.Pods {
			pods[p.Name] = p
		}
	}

	for _, subset := range eps.Subsets {
		for _, addr := range subset.Addresses {
			if addr.TargetRef != nil {
				n := addr.TargetRef.Name
				if p, ok := pods[n]; ok == false {
					return true
				} else {
					p.IsReady = true
				}
			}
		}

		for _, addr := range subset.NotReadyAddresses {
			if addr.TargetRef != nil {
				n := addr.TargetRef.Name
				if p, ok := pods[n]; ok == false {
					return true
				} else {
					p.IsReady = false
				}
			}
		}
	}
	return false
}

func (s *ServiceMonitor) OnNewIngress(p *extv1beta1.Ingress) {
}
func (s *ServiceMonitor) OnUpdateIngress(old, new *extv1beta1.Ingress) {
}
func (s *ServiceMonitor) OnDeleteIngress(p *extv1beta1.Ingress) {
}

func isMapEqual(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k, v1 := range m1 {
		v2, ok := m2[k]
		if ok == false || v1 != v2 {
			return false
		}
	}
	return true
}
