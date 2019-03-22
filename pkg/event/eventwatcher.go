package event

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
)

type EventWatcher struct {
	maxSize   uint
	lock      sync.RWMutex
	eventList *list.List
	events    map[string]map[string]*list.Element
	stopCh    chan struct{}
}

func New(k8sCfg *rest.Config, size uint) (*EventWatcher, error) {
	c, err := cache.New(k8sCfg, cache.Options{})
	if err != nil {
		return nil, fmt.Errorf("create cache failed %v\n", err.Error())
	}

	stop := make(chan struct{})
	go c.Start(stop)
	c.WaitForCacheSync(stop)
	ctrl := controller.New("eventWatcher", c, scheme.Scheme)
	ctrl.Watch(&corev1.Event{})
	ew := &EventWatcher{
		maxSize:   size,
		eventList: list.New(),
		events:    make(map[string]map[string]*list.Element),
		stopCh:    stop,
	}

	go ctrl.Start(stop, ew, predicate.NewIgnoreUnchangedUpdate())
	return ew, nil
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

func (ew *EventWatcher) GetOneNamespaceEvents(namespace string) []*corev1.Event {
	var k8sevents []*corev1.Event
	ew.lock.RLock()
	elems, ok := ew.events[namespace]
	ew.lock.RUnlock()
	if ok {
		for _, elem := range elems {
			k8sevents = append(k8sevents, elem.Value.(*corev1.Event))
		}
	}

	return k8sevents
}

func (ew *EventWatcher) GetAllNamespaceEvents() []*corev1.Event {
	var k8sevents []*corev1.Event
	ew.lock.RLock()
	for _, elems := range ew.events {
		for _, elem := range elems {
			k8sevents = append(k8sevents, elem.Value.(*corev1.Event))
		}
	}
	ew.lock.RUnlock()

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
