package alarm

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	eventReason = "resource shortage"
)

var EventLevelFilter = []string{corev1.EventTypeWarning}
var EventKindFilter = []string{
	"Cluster",
	"Node",
	"Namespace",
	"Pod",
	"StatefulSet",
	"Deployment",
	"DaemonSet",
	"StorageClass",
	"PersistentVolume",
	"PersistentVolumeClaim",
}

type EventCache struct {
	cluster   string
	cache     *AlarmCache
	stopCh    chan struct{}
	startTime int64
}

func NewEventCache(name string, cache cache.Cache, alarmCache *AlarmCache) *EventCache {
	stop := make(chan struct{})
	eventCache := &EventCache{
		cluster:   name,
		stopCh:    stop,
		cache:     alarmCache,
		startTime: time.Now().Unix(),
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
		if ec.startTime < event.LastTimestamp.Time.Unix() && checkEventTypeAndKind(event) {
			alarm := ec.k8sEventToAlarm(event)
			ec.cache.Add(alarm)
		}
	}

	return handler.Result{}, nil
}

func (ec *EventCache) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	if event, ok := e.ObjectNew.(*corev1.Event); ok {
		if ec.startTime < event.LastTimestamp.Time.Unix() && checkEventTypeAndKind(event) {
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
	return event.Reason == eventReason
}

func (ec *EventCache) k8sEventToAlarm(event *corev1.Event) *types.Alarm {
	return &types.Alarm{
		Time:      resource.ISOTime(event.LastTimestamp.Time),
		Type:      types.EventType,
		Cluster:   ec.cluster,
		Namespace: event.InvolvedObject.Namespace,
		Kind:      event.InvolvedObject.Kind,
		Name:      event.InvolvedObject.Name,
		Reason:    event.Reason,
		Message:   event.Message,
	}
}
