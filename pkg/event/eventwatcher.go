package event

import (
	"container/list"
	"sync"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
)

type EventListener struct {
	lastID  uint64
	stopCh  chan struct{}
	eventCh chan *corev1.Event
}

type Event struct {
	id       uint64
	k8sEvent *corev1.Event
}

func (l *EventListener) EventChannel() <-chan *corev1.Event {
	return l.eventCh
}

func (l *EventListener) Stop() {
	l.stopCh <- struct{}{}
	<-l.stopCh
	close(l.eventCh)
}

type EventWatcher struct {
	eventID   uint64
	maxSize   uint
	lock      sync.RWMutex
	cond      *sync.Cond
	eventList *list.List
	stopCh    chan struct{}
}

func New(cache cache.Cache, size uint) (*EventWatcher, error) {
	stop := make(chan struct{})
	ctrl := controller.New("eventWatcher", cache, scheme.Scheme)
	ctrl.Watch(&corev1.Event{})
	ew := &EventWatcher{
		eventID:   1,
		maxSize:   size,
		eventList: list.New(),
		stopCh:    stop,
	}
	ew.cond = sync.NewCond(&ew.lock)

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
	return handler.Result{}, nil
}

func (ew *EventWatcher) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (ew *EventWatcher) add(event *corev1.Event) {
	id := atomic.AddUint64(&ew.eventID, 1)
	e := Event{
		id:       id,
		k8sEvent: event,
	}
	ew.lock.Lock()
	ew.eventList.PushBack(e)
	if uint(ew.eventList.Len()) > ew.maxSize {
		elem := ew.eventList.Front()
		ew.eventList.Remove(elem)
	}
	ew.lock.Unlock()
	ew.cond.Broadcast()
}

func (ew *EventWatcher) AddListener() *EventListener {
	l := &EventListener{
		lastID:  0,
		stopCh:  make(chan struct{}),
		eventCh: make(chan *corev1.Event),
	}

	go ew.publishEvent(l)
	return l
}

func (ew *EventWatcher) publishEvent(l *EventListener) {
	batchSize := ew.maxSize / 4
	events := make([]*corev1.Event, batchSize)
	for {
		lastID, c := ew.getEventsAfterID(l.lastID, events)
		select {
		case <-l.stopCh:
			l.stopCh <- struct{}{}
			return
		default:
		}

		if c == 0 {
			ew.lock.Lock()
			ew.cond.Wait()
			ew.lock.Unlock()
			continue
		}

		l.lastID = lastID
		for i := 0; i < c; i++ {
			select {
			case <-l.stopCh:
				l.stopCh <- struct{}{}
				return
			case l.eventCh <- events[i]:
			}
		}
	}
}

func (ew *EventWatcher) getEventsAfterID(id uint64, events []*corev1.Event) (uint64, int) {
	ew.lock.RLock()
	defer ew.lock.RUnlock()

	elem := ew.eventList.Front()
	if elem == nil {
		return 0, 0
	}

	begID := elem.Value.(Event).id
	if id < begID {
		return ew.getEventsFromOutdated(id, events)
	}

	elem = ew.eventList.Back()
	if elem == nil {
		return 0, 0
	}

	endID := elem.Value.(Event).id
	if id == endID {
		return 0, 0
	}

	if id-begID < endID-id {
		return ew.getEventsFromOutdated(id, events)
	} else {
		return ew.getEventsFromLatest(id, events)
	}
}

func (ew *EventWatcher) getEventsFromOutdated(id uint64, events []*corev1.Event) (uint64, int) {
	elem := ew.eventList.Front()
	for elem.Value.(Event).id <= id {
		elem = elem.Next()
	}
	return ew.getEventsFromElem(elem, events)
}

func (ew *EventWatcher) getEventsFromLatest(id uint64, events []*corev1.Event) (uint64, int) {
	elem := ew.eventList.Back()
	for elem.Value.(Event).id > id {
		elem = elem.Prev()
	}
	elem = elem.Next()
	return ew.getEventsFromElem(elem, events)
}

func (ew *EventWatcher) getEventsFromElem(elem *list.Element, events []*corev1.Event) (uint64, int) {
	ec := 0
	batch := len(events)
	startID := elem.Value.(Event).id
	for elem != nil && ec < batch {
		e := elem.Value.(Event)
		events[ec] = e.k8sEvent
		ec += 1
		elem = elem.Next()
	}
	return startID + uint64(ec-1), ec
}
