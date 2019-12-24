package alarm

import (
	"container/list"
	"sync"
	"sync/atomic"

	"github.com/zdnscloud/cement/slice"
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
	ac := &AlarmCache{
		clusterName: name,
		eventID:     1,
		maxSize:     size,
		alarmList:   list.New(),
		stopCh:      stop,
		ackCh:       make(chan int),
		cli:         cli,
	}
	ac.cond = sync.NewCond(&ac.lock)
	go publishK8sEvent(ac, cache, stop)
	go publishAlarmEvent(ac, stop)
	return ac, nil
}

func (ac *AlarmCache) Stop() {
	close(ac.stopCh)
}

func (al *AlarmListener) AlarmChannel() <-chan Alarm {
	return al.alarmCh
}

func (al *AlarmListener) Stop() {
	al.stopCh <- struct{}{}
	<-al.stopCh
	close(al.alarmCh)
}

func (ac *AlarmCache) AddListener() *AlarmListener {
	al := &AlarmListener{
		lastID:  0,
		stopCh:  make(chan struct{}),
		alarmCh: make(chan Alarm),
	}

	go ac.publishEvent(al)
	return al
}

func (ac *AlarmCache) publishEvent(al *AlarmListener) {
	events := make([]*Alarm, ac.maxSize)
	for {
		lastID, c := ac.getAlarmsAfterID(al.lastID, events)
		select {
		case <-al.stopCh:
			al.stopCh <- struct{}{}
			return
		default:
		}

		if c == 0 {
			ac.lock.Lock()
			ac.cond.Wait()
			ac.lock.Unlock()
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
					ac.unAckNumber -= 1
				}
			}
		}
	}
}

func (ac *AlarmCache) getAlarmsAfterID(id uint64, events []*Alarm) (uint64, int) {
	ac.lock.RLock()
	defer ac.lock.RUnlock()

	elem := ac.alarmList.Front()
	if elem == nil {
		return 0, 0
	}

	begID := elem.Value.(*Alarm).ID
	if id < begID {
		return ac.getAlarmsFromOutdated(id, events)
	}

	elem = ac.alarmList.Back()
	if elem == nil {
		return 0, 0
	}

	endID := elem.Value.(*Alarm).ID
	if id == endID {
		return 0, 0
	}

	if id-begID < endID-id {
		return ac.getAlarmsFromOutdated(id, events)
	} else {
		return ac.getAlarmsFromLatest(id, events)
	}
}

func (ac *AlarmCache) getAlarmsFromOutdated(id uint64, events []*Alarm) (uint64, int) {
	elem := ac.alarmList.Front()
	for elem.Value.(*Alarm).ID <= id {
		elem = elem.Next()
	}
	return ac.getAlarmsFromElem(elem, events)
}

func (ac *AlarmCache) getAlarmsFromLatest(id uint64, events []*Alarm) (uint64, int) {
	elem := ac.alarmList.Back()
	for elem.Value.(*Alarm).ID > id {
		elem = elem.Prev()
	}
	elem = elem.Next()
	return ac.getAlarmsFromElem(elem, events)
}

func (ac *AlarmCache) getAlarmsFromElem(elem *list.Element, events []*Alarm) (uint64, int) {
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

func (ac *AlarmCache) Add(alarm *Alarm) {
	ac.lock.Lock()
	defer ac.lock.Unlock()
	if elem := ac.alarmList.Back(); elem != nil {
		if isRepeat(elem.Value.(*Alarm), alarm) {
			return
		}
	}

	alarm.ID = atomic.AddUint64(&ac.eventID, 1)
	alarm.Cluster = ac.clusterName
	if slice.SliceIndex(ClusterKinds, alarm.Kind) == -1 {
		alarm.Namespace = ""
	}
	ac.alarmList.PushBack(alarm)
	SendMail(ac.cli, alarm)
	addUnAck := true
	if uint(ac.alarmList.Len()) > ac.maxSize {
		elem := ac.alarmList.Front()
		if !elem.Value.(*Alarm).Acknowledged {
			addUnAck = false
		}
		ac.alarmList.Remove(elem)
	}
	if addUnAck {
		ac.unAckNumber += 1
	}
	ac.cond.Broadcast()
}

func isRepeat(lastAlarm, newAlarm *Alarm) bool {
	if lastAlarm.Namespace == newAlarm.Namespace && lastAlarm.Kind == newAlarm.Kind && lastAlarm.Message == newAlarm.Message && lastAlarm.Name == newAlarm.Name {
		return true
	}
	return false
}
