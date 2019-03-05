package exec

import (
	"sync"

	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	corev1 "k8s.io/api/core/v1"
)

type notifyTask struct {
	Namespace string
	PodName   string
	Ready     chan<- struct{}
}

type podWatcher struct {
	knownPods map[string]map[string]struct{}

	mu          sync.Mutex
	notifyTasks []notifyTask
}

func newPodWatcher() *podWatcher {
	return &podWatcher{
		knownPods: make(map[string]map[string]struct{}),
	}
}

func (w *podWatcher) OnCreate(e event.CreateEvent) (handler.Result, error) {
	if p, ok := e.Object.(*corev1.Pod); ok {
		if p.Status.Phase == corev1.PodRunning {
			w.mu.Lock()
			w.addReadyPod(p)
			w.mu.Unlock()
		}
	}
	return handler.Result{}, nil
}

func (w *podWatcher) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	if p, ok := e.ObjectNew.(*corev1.Pod); ok {
		o := e.ObjectOld.(*corev1.Pod)
		if o.Status.Phase != p.Status.Phase {
			w.mu.Lock()
			if p.Status.Phase == corev1.PodRunning {
				w.addReadyPod(p)
			} else if o.Status.Phase == corev1.PodRunning {
				w.removeUnreadyPod(p)
			}
			w.mu.Unlock()
		}
	}
	return handler.Result{}, nil
}

func (w *podWatcher) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	if p, ok := e.Object.(*corev1.Pod); ok {
		w.mu.Lock()
		w.removeUnreadyPod(p)
		w.mu.Unlock()
	}
	return handler.Result{}, nil
}

func (w *podWatcher) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (w *podWatcher) AddNotifyTask(namespace, podName string) <-chan struct{} {
	ready := make(chan struct{}, 1)
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isPodReady(namespace, podName) {
		ready <- struct{}{}
	} else {
		w.notifyTasks = append(w.notifyTasks, notifyTask{Namespace: namespace, PodName: podName, Ready: ready})
	}
	return ready
}

func (w *podWatcher) addReadyPod(p *corev1.Pod) {
	pods, ok := w.knownPods[p.Namespace]
	if ok == false {
		pods = make(map[string]struct{})
		w.knownPods[p.Namespace] = pods
	}
	pods[p.Name] = struct{}{}
	w.notifyReadyPod()
}

func (w *podWatcher) removeUnreadyPod(p *corev1.Pod) {
	pods, ok := w.knownPods[p.Namespace]
	if ok == false {
		return
	}
	delete(pods, p.Name)
}

func (w *podWatcher) isPodReady(namespace, podName string) bool {
	pods, ok := w.knownPods[namespace]
	if ok == false {
		return false
	}
	_, ok = pods[podName]
	return ok
}

func (w *podWatcher) notifyReadyPod() {
	tasks := make([]notifyTask, 0, len(w.notifyTasks))
	for _, t := range w.notifyTasks {
		if w.isPodReady(t.Namespace, t.PodName) {
			t.Ready <- struct{}{}
		} else {
			tasks = append(tasks, t)
		}
	}
	w.notifyTasks = tasks
}
