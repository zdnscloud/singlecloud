package alarm

import (
	"container/list"
	"sync"
	"sync/atomic"

	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/cement/uuid"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
)

type AlarmCache struct {
	clusterName string
	eventID     uint64
	maxSize     uint
	lock        sync.RWMutex
	cond        *sync.Cond
	alarmList   *list.List
	stopCh      chan struct{}
	unAckNumber int
	ackCh       chan int
	cli         client.Client
}

type AlarmListener struct {
	lastID  uint64
	stopCh  chan struct{}
	alarmCh chan Alarm
}

func NewAlarmCache(cache cache.Cache, cli client.Client, size uint, name string) (*AlarmCache, error) {
	stop := make(chan struct{})
	aw := &AlarmCache{
		clusterName: name,
		eventID:     1,
		maxSize:     size,
		alarmList:   list.New(),
		stopCh:      stop,
		ackCh:       make(chan int),
		cli:         cli,
	}
	aw.cond = sync.NewCond(&aw.lock)
	go publishK8sEvent(aw, cache, stop)
	go publishAlarmEvent(aw, stop)
	return aw, nil
}

func (aw *AlarmCache) Stop() {
	close(aw.stopCh)
}

func (al *AlarmListener) AlarmChannel() <-chan Alarm {
	return al.alarmCh
}

func (al *AlarmListener) Stop() {
	al.stopCh <- struct{}{}
	<-al.stopCh
	close(al.alarmCh)
}

func (aw *AlarmCache) AddListener() *AlarmListener {
	al := &AlarmListener{
		lastID:  0,
		stopCh:  make(chan struct{}),
		alarmCh: make(chan Alarm),
	}

	go aw.publishEvent(al)
	return al
}

func (aw *AlarmCache) publishEvent(al *AlarmListener) {
	events := make([]*Alarm, aw.maxSize)
	for {
		lastID, c := aw.getAlarmsAfterID(al.lastID, events)
		select {
		case <-al.stopCh:
			al.stopCh <- struct{}{}
			return
		default:
		}

		if c == 0 {
			aw.lock.Lock()
			aw.cond.Wait()
			aw.lock.Unlock()
			continue
		}

		al.lastID = lastID
		for i := 0; i < c; i++ {
			select {
			case <-al.stopCh:
				al.stopCh <- struct{}{}
				return
			case al.alarmCh <- *events[i]:
				if !events[i].Acknowledged {
					events[i].Acknowledged = true
					aw.unAckNumber -= 1
				}
			}
		}
	}
}

func (aw *AlarmCache) getAlarmsAfterID(id uint64, events []*Alarm) (uint64, int) {
	aw.lock.RLock()
	defer aw.lock.RUnlock()

	elem := aw.alarmList.Front()
	if elem == nil {
		return 0, 0
	}

	begID := elem.Value.(*Alarm).ID
	if id < begID {
		return aw.getAlarmsFromOutdated(id, events)
	}

	elem = aw.alarmList.Back()
	if elem == nil {
		return 0, 0
	}

	endID := elem.Value.(*Alarm).ID
	if id == endID {
		return 0, 0
	}

	if id-begID < endID-id {
		return aw.getAlarmsFromOutdated(id, events)
	} else {
		return aw.getAlarmsFromLatest(id, events)
	}
}

func (aw *AlarmCache) getAlarmsFromOutdated(id uint64, events []*Alarm) (uint64, int) {
	elem := aw.alarmList.Front()
	for elem.Value.(*Alarm).ID <= id {
		elem = elem.Next()
	}
	return aw.getAlarmsFromElem(elem, events)
}

func (aw *AlarmCache) getAlarmsFromLatest(id uint64, events []*Alarm) (uint64, int) {
	elem := aw.alarmList.Back()
	for elem.Value.(*Alarm).ID > id {
		elem = elem.Prev()
	}
	elem = elem.Next()
	return aw.getAlarmsFromElem(elem, events)
}

func (aw *AlarmCache) getAlarmsFromElem(elem *list.Element, events []*Alarm) (uint64, int) {
	ec := 0
	batch := len(events)
	startID := elem.Value.(*Alarm).ID
	for elem != nil && ec < batch {
		events[ec] = elem.Value.(*Alarm)
		ec += 1
		elem = elem.Next()
	}
	return startID + uint64(ec-1), ec
}

func (aw *AlarmCache) Add(alarm *Alarm) {
	aw.lock.Lock()
	var repeat bool
	elem := aw.alarmList.Back()
	if elem != nil {
		lastone := elem.Value.(*Alarm)
		if lastone.Namespace == alarm.Namespace && lastone.Kind == alarm.Kind && lastone.Message == alarm.Message && lastone.Name == alarm.Name {
			repeat = true
		}
	}

	if !repeat {
		alarm.ID = atomic.AddUint64(&aw.eventID, 1)
		uid, _ := uuid.Gen()
		alarm.UUID = uid
		alarm.Cluster = aw.clusterName
		if slice.SliceIndex(ClusterKinds, alarm.Kind) == -1 {
			alarm.Namespace = ""
		}
		aw.alarmList.PushBack(alarm)
		SendMail(aw.cli, alarm)
		addUnAck := true
		if uint(aw.alarmList.Len()) > aw.maxSize {
			elem := aw.alarmList.Front()

			if !elem.Value.(*Alarm).Acknowledged {
				addUnAck = false
			}
			aw.alarmList.Remove(elem)
		}
		if addUnAck {
			aw.unAckNumber += 1
		}
	}
	aw.cond.Broadcast()
	aw.lock.Unlock()
}
