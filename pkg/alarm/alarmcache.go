package alarm

import (
	"container/list"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/db"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type AlarmCache struct {
	eventID        uint64
	maxSize        uint
	lock           sync.RWMutex
	cond           *sync.Cond
	alarmList      *list.List
	ackList        *list.List
	stopCh         chan struct{}
	unAckNumber    uint64
	ackCh          chan int
	ThresholdTable kvzoo.Table
}

type AlarmListener struct {
	lastID  uint64
	stopCh  chan struct{}
	alarmCh chan interface{}
}

func NewAlarmCache(size uint) (*AlarmCache, error) {
	tn, _ := kvzoo.TableNameFromSegments(types.ThresholdTable)
	table, err := db.GetGlobalDB().CreateOrGetTable(tn)
	if err != nil {
		return nil, fmt.Errorf("create or get table %s failed: %s", types.ThresholdTable, err.Error())
	}
	stop := make(chan struct{})
	ac := &AlarmCache{
		eventID:        0,
		maxSize:        size,
		alarmList:      list.New(),
		ackList:        list.New(),
		stopCh:         stop,
		ackCh:          make(chan int),
		ThresholdTable: table,
	}
	ac.cond = sync.NewCond(&ac.lock)
	go subscribeAlarmEvent(ac, stop)
	return ac, nil
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
		lastID:  ac.eventID,
		stopCh:  make(chan struct{}),
		alarmCh: make(chan interface{}),
	}

	go ac.publishEvent(al)
	go ac.publishAck(al)
	return al
}

func (ac *AlarmCache) publishEvent(al *AlarmListener) {
	for {
		select {
		case <-al.stopCh:
			al.stopCh <- struct{}{}
			return
		default:
		}
		alarms := ac.getAlarmsAfterID(al.lastID)
		c := len(alarms)

		if c == 0 {
			ac.lock.Lock()
			ac.cond.Wait()
			ac.lock.Unlock()
			continue
		}
		al.lastID += uint64(c)
		for _, alarm := range alarms {
			select {
			case <-al.stopCh:
				al.stopCh <- struct{}{}
				return
			case al.alarmCh <- *alarm:
			}
		}
	}
}

func (ac *AlarmCache) getAlarmsAfterID(id uint64) []*types.Alarm {
	ac.lock.RLock()
	defer ac.lock.RUnlock()
	var alarms types.Alarms
	if ac.alarmList.Len() == 0 || id == ac.eventID {
		return alarms
	}
	for e := ac.alarmList.Back(); e != nil; e = e.Prev() {
		alarm := e.Value.(*types.Alarm)
		if alarm.UID <= id {
			break
		}
		alarms = append(alarms, alarm)
	}
	return alarms
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
	if slice.SliceIndex(ClusterKinds, alarm.Kind) >= 0 {
		alarm.Namespace = ""
	}
	if err := SendMail(alarm, ac.ThresholdTable); err != nil {
		log.Warnf("send mail failed: %s", err)
	}
	ac.alarmList.PushBack(alarm)
	if uint(ac.alarmList.Len()) > ac.maxSize {
		elem := ac.alarmList.Front()
		ac.alarmList.Remove(elem)
	} else {
		ac.SetUnAck(1)
	}
	ac.cond.Broadcast()
}

func (ac *AlarmCache) SetUnAck(u int) {
	atomic.AddUint64(&ac.unAckNumber, uint64(u))
}

func isRepeat(lastAlarm, newAlarm *types.Alarm) bool {
	return lastAlarm.Cluster == newAlarm.Cluster &&
		lastAlarm.Namespace == newAlarm.Namespace &&
		lastAlarm.Kind == newAlarm.Kind &&
		lastAlarm.Reason == newAlarm.Reason &&
		lastAlarm.Message == newAlarm.Message &&
		lastAlarm.Name == newAlarm.Name
}

func (ac *AlarmCache) ackListAdd(alarm *types.Alarm) {
	if ac.ackList.Len() == 0 {
		ac.ackList.PushBack(alarm)
		return
	}
	if e := ac.ackList.Front(); alarm.UID < e.Value.(*types.Alarm).UID {
		ac.ackList.PushFront(alarm)
		return
	}
	for e := ac.ackList.Back(); e != nil; e = e.Prev() {
		if alarm.UID > e.Value.(*types.Alarm).UID {
			ac.ackList.InsertAfter(alarm, e)
			return
		}
	}
}
