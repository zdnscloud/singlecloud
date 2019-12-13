package alarm

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/zdnscloud/gok8s/cache"
)

func NewAlarmWatcher(cache cache.Cache, size uint, ZcloudEventCh <-chan interface{}) (*AlarmWatcher, error) {
	stop := make(chan struct{})
	aw := &AlarmWatcher{
		eventID:       1,
		maxSize:       size,
		alarmList:     list.New(),
		stopCh:        stop,
		ackCh:         make(chan int),
		zcloudEventCh: ZcloudEventCh,
	}
	aw.cond = sync.NewCond(&aw.lock)

	go publishK8sEvent(aw, cache, stop)
	go publishZloudEvent(aw, stop)
	return aw, nil
}

func (aw *AlarmWatcher) Stop() {
	close(aw.stopCh)
}

func (aw *AlarmWatcher) Add(alarm *Alarm) {
	aw.lock.Lock()
	aw.alarmList.PushBack(alarm)
	if uint(aw.alarmList.Len()) > aw.maxSize {
		elem := aw.alarmList.Front()
		aw.alarmList.Remove(elem)
	}
	fmt.Println("in one messgae, before", aw.unAckNumber, "atfer ", aw.unAckNumber+1)
	aw.unAckNumber += 1
	aw.lock.Unlock()
	aw.cond.Broadcast()
}

func (aw *AlarmWatcher) AddListener() *AlarmListener {
	al := &AlarmListener{
		lastID:  0,
		stopCh:  make(chan struct{}),
		alarmCh: make(chan Alarm),
	}

	go aw.publishEvent(al)
	return al
}

func (aw *AlarmWatcher) publishEvent(al *AlarmListener) {
	batchSize := aw.maxSize / 4
	events := make([]*Alarm, batchSize)
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
				fmt.Println("!!!!!send msg to resv", events[i])
				if !events[i].Acknowledged {
					events[i].Acknowledged = true
					fmt.Println("out one messgae, before", aw.unAckNumber, "atfer ", aw.unAckNumber-1)
					aw.unAckNumber -= 1
				}
			}
		}
	}
}

func (aw *AlarmWatcher) getAlarmsAfterID(id uint64, events []*Alarm) (uint64, int) {
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

func (aw *AlarmWatcher) getAlarmsFromOutdated(id uint64, events []*Alarm) (uint64, int) {
	elem := aw.alarmList.Front()
	for elem.Value.(*Alarm).ID <= id {
		elem = elem.Next()
	}
	return aw.getAlarmsFromElem(elem, events)
}

func (aw *AlarmWatcher) getAlarmsFromLatest(id uint64, events []*Alarm) (uint64, int) {
	elem := aw.alarmList.Back()
	for elem.Value.(*Alarm).ID > id {
		elem = elem.Prev()
	}
	elem = elem.Next()
	return aw.getAlarmsFromElem(elem, events)
}

func (aw *AlarmWatcher) getAlarmsFromElem(elem *list.Element, events []*Alarm) (uint64, int) {
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

func (al *AlarmListener) Stop() {
	al.stopCh <- struct{}{}
	<-al.stopCh
	close(al.alarmCh)
}

func (al *AlarmListener) AlarmChannel() <-chan Alarm {
	return al.alarmCh
}
