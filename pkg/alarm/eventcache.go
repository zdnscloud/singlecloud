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
)

func publishK8sEvent(ac *AlarmCache, cache cache.Cache, stop chan struct{}) {
	ctrl := controller.New("eventWatcher", cache, scheme.Scheme)
	ctrl.Watch(&corev1.Event{})
	go ctrl.Start(stop, ac, predicate.NewIgnoreUnchangedUpdate())
}

func (ac *AlarmCache) OnCreate(e event.CreateEvent) (handler.Result, error) {
	if event, ok := e.Object.(*corev1.Event); ok {
		if checkEventTypeAndKind(event) {
			ac.Add(k8sEventToAlarm(event))
		}
	}

	return handler.Result{}, nil
}

func (ac *AlarmCache) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	if event, ok := e.ObjectNew.(*corev1.Event); ok {
		if checkEventTypeAndKind(event) {
			ac.Add(k8sEventToAlarm(event))
		}
	}

	return handler.Result{}, nil
}

func (ac *AlarmCache) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (ac *AlarmCache) OnGeneric(e event.GenericEvent) (handler.Result, error) {
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

func k8sEventToAlarm(event *corev1.Event) *Alarm {
	t := event.LastTimestamp
	return &Alarm{
		Time:      fmt.Sprintf("%.2d:%.2d:%.2d", t.Hour(), t.Minute(), t.Second()),
		Type:      EventType,
		Namespace: event.InvolvedObject.Namespace,
		Kind:      event.InvolvedObject.Kind,
		Name:      event.InvolvedObject.Name,
		Reason:    event.Reason,
		Message:   event.Message,
	}
}
