package k8seventwatcher

import (
	"container/list"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ut "github.com/zdnscloud/cement/unittest"
	corev1 "k8s.io/api/core/v1"
)

var logFile = "acl.log"

type dumbListener struct {
	eventCh    <-chan *corev1.Event
	eventCount uint32
	lastID     uint32
}

func newDumbListener(l *EventListener, t *testing.T) *dumbListener {
	dl := &dumbListener{
		eventCh: l.EventChannel(),
	}
	go dl.recvEvents(t)
	return dl
}

func (l *dumbListener) recvEvents(t *testing.T) {
	lastCount := int32(0)
	for {
		e, ok := <-l.eventCh
		if ok == false {
			return
		}
		//fmt.Printf("---> %v\n", e.Count)

		if lastCount != 0 {
			ut.Equal(t, lastCount+1, e.Count)
		}
		lastCount = e.Count
		atomic.AddUint32(&l.eventCount, 1)
	}
}

func (l *dumbListener) getEventCount() uint32 {
	return atomic.LoadUint32(&l.eventCount)
}

func TestEventWatch(t *testing.T) {
	maxSize := 128
	ew := &EventWatcher{
		eventID:   1,
		maxSize:   uint(maxSize),
		eventList: list.New(),
		stopCh:    make(chan struct{}),
	}
	ew.cond = sync.NewCond(&ew.lock)

	l1 := ew.AddListener()
	l2 := ew.AddListener()

	dl1 := newDumbListener(l1, t)
	dl2 := newDumbListener(l2, t)
	addEvents := func(beg, end int) {
		for i := beg; i < end; i++ {
			ew.add(&corev1.Event{
				Count: int32(i),
			})
		}
	}

	beg := 0
	batch := maxSize/2 - 1
	for i := 0; i < 10; i++ {
		addEvents(beg, beg+batch)
		beg = beg + batch
		<-time.After(1 * time.Second)
	}
	<-time.After(2 * time.Second)
	ut.Equal(t, dl1.getEventCount(), uint32(beg))
	ut.Equal(t, dl2.getEventCount(), uint32(beg))
}
