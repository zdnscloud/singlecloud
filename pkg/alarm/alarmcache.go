package alarm

import (
	"container/list"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

type AlarmCache struct {
	eventID     uint64
	maxSize     uint
	lock        sync.RWMutex
	cond        *sync.Cond
	alarmList   *list.List
	stopCh      chan struct{}
	unAckNumber uint64
	ackCh       chan int
	clusters    map[string]*zke.Cluster
}

type AlarmListener struct {
	lastID  uint64
	stopCh  chan struct{}
	alarmCh chan interface{}
}

func NewAlarmCache(size uint, clusters map[string]*zke.Cluster) *AlarmCache {
	stop := make(chan struct{})
	ac := &AlarmCache{
		eventID:   0,
		maxSize:   size,
		alarmList: list.New(),
		stopCh:    stop,
		ackCh:     make(chan int),
		clusters:  clusters,
	}
	ac.cond = sync.NewCond(&ac.lock)
	go subscribeAlarmEvent(ac, stop)
	return ac
}

func (ac *AlarmCache) Stop() {
	close(ac.stopCh)
}

func (al *AlarmListener) AlarmChannel() <-chan interface{} {
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
		alarmCh: make(chan interface{}),
	}

	go ac.publishEvent(al)
	go ac.publishAck(al)
	return al
}

func (ac *AlarmCache) publishEvent(al *AlarmListener) {
	events := make([]*types.Alarm, ac.maxSize)
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
			if events[i].Acknowledged {
				continue
			}
			select {
			case <-al.stopCh:
				al.stopCh <- struct{}{}
				return
			case al.alarmCh <- *events[i]:
			}
		}
	}
}

func (ac *AlarmCache) getAlarmsAfterID(id uint64, events []*types.Alarm) (uint64, int) {
	ac.lock.RLock()
	defer ac.lock.RUnlock()
	if ac.alarmList.Len() == int(id) {
		return 0, 0
	}
	if id == 0 {
		return ac.getAlarmsFromOutdated(id, events)
	} else {
		return ac.getAlarmsFromLatest(id, events)
	}
}

func (ac *AlarmCache) getAlarmsFromOutdated(id uint64, events []*types.Alarm) (uint64, int) {
	elem := ac.alarmList.Front()
	for elem.Value.(*types.Alarm).UID <= id {
		elem = elem.Next()
	}
	return ac.getAlarmsFromElem(elem, events)
}

func (ac *AlarmCache) getAlarmsFromLatest(id uint64, events []*types.Alarm) (uint64, int) {
	elem := ac.alarmList.Back()
	for elem.Value.(*types.Alarm).UID > id {
		elem = elem.Prev()
	}
	elem = elem.Next()
	return ac.getAlarmsFromElem(elem, events)
}

func (ac *AlarmCache) getAlarmsFromElem(elem *list.Element, events []*types.Alarm) (uint64, int) {
	ec := 0
	batch := len(events)
	startID := elem.Value.(*types.Alarm).UID
	for elem != nil && ec < batch {
		events[ec] = elem.Value.(*types.Alarm)
		ec += 1
		elem = elem.Next()
	}
	return startID + uint64(ec-1), ec
}

func (ac *AlarmCache) Add(alarm *types.Alarm) {
	ac.lock.Lock()
	defer ac.lock.Unlock()
	if elem := ac.alarmList.Back(); elem != nil {
		if isRepeat(elem.Value.(*types.Alarm), alarm) {
			return
		}
	}
	alarm.UID = atomic.AddUint64(&ac.eventID, 1)
	alarm.SetID(strconv.Itoa(int(alarm.UID)))
	cluster, ok := ac.clusters[alarm.Cluster]
	if ok {
		SendMail(cluster.KubeClient, alarm)
	}
	if slice.SliceIndex(ClusterKinds, alarm.Kind) == -1 {
		alarm.Namespace = ""
	}
	ac.alarmList.PushBack(alarm)
	addUnAck := true
	if uint(ac.alarmList.Len()) > ac.maxSize {
		elem := ac.alarmList.Front()
		if !elem.Value.(*types.Alarm).Acknowledged {
			addUnAck = false
		}
		ac.alarmList.Remove(elem)
	}
	if addUnAck {
		ac.SetUnAck(1)
	}
	ac.cond.Broadcast()
}

func (ac *AlarmCache) SetUnAck(u int) {
	atomic.AddUint64(&ac.unAckNumber, uint64(u))
}

func isRepeat(lastAlarm, newAlarm *types.Alarm) bool {
	if lastAlarm.Namespace == newAlarm.Namespace && lastAlarm.Kind == newAlarm.Kind && lastAlarm.Message == newAlarm.Message && lastAlarm.Name == newAlarm.Name {
		return true
	}
	return false
}
