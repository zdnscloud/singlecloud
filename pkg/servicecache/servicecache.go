package resourcerepo

import (
	"context"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	"github.com/zdnscloud/singlecloud/pkg/logger"
)

type ResourceRepo struct {
	services map[string]*ServiceMonitor
	lock     sync.Mutex
	cache    cache.Cache
	stopCh   chan struct{}
}

func New(k8sCfg *rest.Config) (*ResourceRepo, error) {
	c, err := cache.New(k8sCfg, cache.Options{})
	if err != nil {
		return nil, fmt.Errorf("create cache failed %v\n", err.Error())
	}

	stopCh := make(chan struct{})
	go c.Start(stopCh)
	c.WaitForCacheSync(stopCh)

	ctrl := controller.New("resourceRepo", c, scheme.Scheme)
	ctrl.Watch(&corev1.Namespace{})
	ctrl.Watch(&corev1.Service{})
	ctrl.Watch(&corev1.Endpoints{})
	ctrl.Watch(&extv1beta1.Ingress{})

	repo := &ResourceRepo{
		stopCh: stopCh,
		cache:  c,
	}
	if err := repo.initServices(); err != nil {
		return nil, err
	}

	go ctrl.Start(stopCh, repo, predicate.NewIgnoreUnchangedUpdate())
	return repo, nil
}

func (r *ResourceRepo) initServices() error {
	nses := &corev1.NamespaceList{}
	err := r.cache.List(context.TODO(), nil, nses)
	if err != nil {
		return err
	}

	services := make(map[string]*ServiceMonitor)
	for _, ns := range nses.Items {
		s := newServiceMonitor(r.cache)
		services[ns.Name] = s
	}
	r.services = services
	return nil
}

func (r *ResourceRepo) GetServices() map[string][]*Service {
	r.lock.Lock()
	defer r.lock.Unlock()

	svcs := make(map[string][]*Service)
	for n, s := range r.services {
		svcs[n] = s.GetServices()
	}
	return svcs
}

func (r *ResourceRepo) OnCreate(e event.CreateEvent) (handler.Result, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Namespace:
		if _, ok := r.services[obj.Name]; ok == false {
			s := newServiceMonitor(r.cache)
			r.services[obj.Name] = s
		}
	case *corev1.Service:
		s, ok := r.services[obj.Namespace]
		if ok == false {
			logger.Error("namespace %s is unknown", obj.Namespace)
		} else {
			s.OnNewService(obj)
		}
	case *extv1beta1.Ingress:
		s, ok := r.services[obj.Namespace]
		if ok == false {
			logger.Error("namespace %s is unknown", obj.Namespace)
		} else {
			s.OnNewIngress(obj)
		}
	}

	return handler.Result{}, nil
}

func (r *ResourceRepo) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	switch newObj := e.ObjectNew.(type) {
	case *corev1.Service:
		s, ok := r.services[newObj.Namespace]
		if ok == false {
			logger.Error("namespace %s is unknown", newObj.Namespace)
		} else {
			s.OnUpdateService(e.ObjectOld.(*corev1.Service), newObj)
		}
	case *corev1.Endpoints:
		s, ok := r.services[newObj.Namespace]
		if ok == false {
			logger.Error("namespace %s is unknown", newObj.Namespace)
		} else {
			s.OnUpdateEndpoints(e.ObjectOld.(*corev1.Endpoints), newObj)
		}
	case *extv1beta1.Ingress:
		s, ok := r.services[newObj.Namespace]
		if ok == false {
			logger.Error("namespace %s is unknown", newObj.Namespace)
		} else {
			s.OnUpdateIngress(e.ObjectOld.(*extv1beta1.Ingress), newObj)
		}
	}

	return handler.Result{}, nil
}

func (r *ResourceRepo) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Namespace:
		_, ok := r.services[obj.Name]
		if ok == false {
			logger.Warn("namespace %s isn't included in repo", obj.Name)
		} else {
			delete(r.services, obj.Name)
		}
	case *corev1.Service:
		s, ok := r.services[obj.Namespace]
		if ok == false {
			logger.Error("namespace %s is unknown", obj.Namespace)
		} else {
			s.OnDeleteService(obj)
		}
	case *extv1beta1.Ingress:
		s, ok := r.services[obj.Namespace]
		if ok == false {
			logger.Error("namespace %s is unknown", obj.Namespace)
		} else {
			s.OnDeleteIngress(obj)
		}
	}

	return handler.Result{}, nil
}

func (r *ResourceRepo) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}
