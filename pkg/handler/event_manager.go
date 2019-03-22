package handler

import (
	"container/list"
	"fmt"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"

	"github.com/zdnscloud/singlecloud/pkg/logger"
)

const defaultEventMaxSize = 10000

type EventManager struct {
	watchers map[string]*EventWatcher
}

func newEventManager() *EventManager {
	return &EventManager{
		watchers: make(map[string]*EventWatcher),
	}
}

func (m *EventManager) AddEventWatcher(clusterID string, conf *rest.Config, size uint) error {
	watcher, err := newEventWatcher(conf, size)
	if err != nil {
		return err
	}
	m.watchers[clusterID] = watcher
	go watcher.Run()
	return nil
}

type EventWatcher struct {
	cache     cache.Cache
	maxSize   uint
	lock      sync.RWMutex
	eventList *list.List
	events    map[string]map[string]*list.Element
}

func newEventWatcher(conf *rest.Config, size uint) (*EventWatcher, error) {
	c, err := cache.New(conf, cache.Options{})
	if err != nil {
		return nil, fmt.Errorf("create cache failed %v\n", err.Error())
	}

	return &EventWatcher{
		cache:     c,
		maxSize:   size,
		eventList: list.New(),
		events:    make(map[string]map[string]*list.Element),
	}, nil
}

func (ew *EventWatcher) Run() {
	stop := make(chan struct{})
	defer close(stop)

	go ew.cache.Start(stop)
	ew.cache.WaitForCacheSync(stop)

	ctrl := controller.New("eventWatcher", ew.cache, scheme.Scheme)
	ctrl.Watch(&corev1.Event{})
	ctrl.Start(stop, ew, predicate.NewIgnoreUnchangedUpdate())
}

func (ew *EventWatcher) OnCreate(e event.CreateEvent) (handler.Result, error) {
	if event, ok := e.Object.(*corev1.Event); ok {
		ew.add(event)
	}

	return handler.Result{}, nil
}

func (ew *EventWatcher) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	if event, ok := e.ObjectNew.(*corev1.Event); ok {
		ew.add(event)
	}
	return handler.Result{}, nil
}

func (ew *EventWatcher) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	if event, ok := e.Object.(*corev1.Event); ok {
		ew.add(event)
	}

	return handler.Result{}, nil
}

func (ew *EventWatcher) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	if event, ok := e.Object.(*corev1.Event); ok {
		ew.add(event)
	}

	return handler.Result{}, nil
}

func (ew *EventWatcher) add(event *corev1.Event) {
	ew.lock.Lock()
	defer ew.lock.Unlock()
	namespace := event.InvolvedObject.Namespace
	key := getEventKey(event)
	k8sevents, ok := ew.events[namespace]
	if ok == false {
		k8sevents = make(map[string]*list.Element)
		ew.events[namespace] = k8sevents
	}

	if elem, ok := k8sevents[key]; ok {
		ew.eventList.MoveToFront(elem)
		elem.Value = event
	} else {
		elem := ew.eventList.PushFront(event)
		k8sevents[key] = elem
	}

	if uint(ew.eventList.Len()) > ew.maxSize {
		elem := ew.eventList.Back()
		if elem != nil {
			ew.eventList.Remove(elem)
			event := elem.Value.(*corev1.Event)
			delete(ew.events[event.InvolvedObject.Namespace], getEventKey(event))
		}
	}
}

func (m *ClusterManager) GetEvents(id, namespace string) interface{} {
	if watcher, ok := m.eventManager.watchers[id]; ok {
		return watcher.getEvents(namespace)
	} else {
		logger.Warn("cluster %s isn't found", id)
		return nil
	}
}

func (ew *EventWatcher) getEvents(namespace string) interface{} {
	ew.lock.RLock()
	defer ew.lock.RUnlock()
	if namespace != "" {
		return ew.getOneNamespaceEvents(namespace)
	} else {
		return ew.getAllNamespaceEvents()
	}
}

func (ew *EventWatcher) getOneNamespaceEvents(namespace string) []*corev1.Event {
	var k8sevents []*corev1.Event
	if elems, ok := ew.events[namespace]; ok {
		for _, elem := range elems {
			k8sevents = append(k8sevents, elem.Value.(*corev1.Event))
		}
	}

	return k8sevents
}

func (ew *EventWatcher) getAllNamespaceEvents() []*corev1.Event {
	var k8sevents []*corev1.Event
	for _, elems := range ew.events {
		for _, elem := range elems {
			k8sevents = append(k8sevents, elem.Value.(*corev1.Event))
		}
	}

	return k8sevents
}

func getEventKey(event *corev1.Event) string {
	return strings.Join([]string{
		event.Source.Component,
		event.Source.Host,
		event.InvolvedObject.Kind,
		event.InvolvedObject.Namespace,
		event.InvolvedObject.Name,
		event.InvolvedObject.FieldPath,
		string(event.InvolvedObject.UID),
		event.InvolvedObject.APIVersion,
		event.Type,
		event.Reason,
		event.Message,
	}, "")
}
