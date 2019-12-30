package alarm

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type EventCache struct {
	cluster string
	cache   *AlarmCache
	stopCh  chan struct{}
}

func NewEventCache(name string, cache cache.Cache, alarmCache *AlarmCache) *EventCache {
	stop := make(chan struct{})
	eventCache := &EventCache{
		cluster: name,
		stopCh:  stop,
		cache:   alarmCache,
	}
	ctrl := controller.New("eventWatcher", cache, scheme.Scheme)
	ctrl.Watch(&corev1.Event{})
	go ctrl.Start(stop, eventCache, predicate.NewIgnoreUnchangedUpdate())
	return eventCache
}

func (ec *EventCache) Stop() {
	close(ec.stopCh)
}

func (ec *EventCache) OnCreate(e event.CreateEvent) (handler.Result, error) {
	if event, ok := e.Object.(*corev1.Event); ok {
		if checkEventTypeAndKind(event) {
			alarm := ec.k8sEventToAlarm(event)
			ec.cache.Add(alarm)
		}
	}

	return handler.Result{}, nil
}

func (ec *EventCache) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	if event, ok := e.ObjectNew.(*corev1.Event); ok {
		if checkEventTypeAndKind(event) {
			alarm := ec.k8sEventToAlarm(event)
			ec.cache.Add(alarm)
		}
	}

	return handler.Result{}, nil
}

func (ec *EventCache) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (ec *EventCache) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func checkEventTypeAndKind(event *corev1.Event) bool {
	if event.Type != corev1.EventTypeNormal {
		switch event.InvolvedObject.Kind {
		case "Pod", "StorageClass", "Cluster", "Namespace", "StatefulSet", "Deployment", "DaemonSet", "PersistentVolume", "PersistentVolumeClaim", "Node":
			return true
		}
	}
	return false
}

func (ec *EventCache) k8sEventToAlarm(event *corev1.Event) *types.Alarm {
	t := event.LastTimestamp
	return &types.Alarm{
		Time:      fmt.Sprintf("%.2d:%.2d:%.2d", t.Hour(), t.Minute(), t.Second()),
		Type:      types.EventType,
		Cluster:   ec.cluster,
		Namespace: event.InvolvedObject.Namespace,
		Kind:      event.InvolvedObject.Kind,
		Name:      event.InvolvedObject.Name,
		Reason:    event.Reason,
		Message:   event.Message,
	}
}
