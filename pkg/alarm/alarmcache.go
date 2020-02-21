package alarm

import (
	"container/list"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AlarmsTableName = "alarms"
)

type AlarmCache struct {
	lock           sync.RWMutex
	cond           *sync.Cond
	stopCh         chan struct{}
	thresholdTable kvzoo.Table
	alarmsTable    kvzoo.Table
	alarmList      *list.List
	unAckNumber    uint64
	eventID        uint64
}

type AlarmListener struct {
	lastID  uint64
	stopCh  chan struct{}
	alarmCh chan interface{}
}

func NewAlarmCache() (*AlarmCache, error) {
	thresholdTable, err := genTable(types.ThresholdTable)
	if err != nil {
		return nil, err
	}
	alarmsTable, err := genTable(AlarmsTableName)
	if err != nil {
		return nil, err
	}
	stop := make(chan struct{})
	ac := &AlarmCache{
		stopCh:         stop,
		thresholdTable: thresholdTable,
		alarmsTable:    alarmsTable,
		alarmList:      list.New(),
	}
	if err := ac.initFromDB(); err != nil {
		return nil, err
	}
	ac.cond = sync.NewCond(&ac.lock)
	go subscribeAlarmEvent(ac, stop)
	return ac, nil
}

func (ac *AlarmCache) initFromDB() error {
	alarms, err := getAlarmsFromDB(ac.alarmsTable)
	if err != nil {
		return err
	}
	for _, alarm := range alarms {
		ac.alarmList.PushBack(alarm)
		if !alarm.Acknowledged {
			ac.unAckNumber += 1
		}
	}
	if ac.alarmList.Len() > 0 {
		ac.eventID = ac.alarmList.Back().Value.(*types.Alarm).UID
	}
	return nil
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
			case al.alarmCh <- alarm:
			}
		}
	}
}

func (ac *AlarmCache) getAlarmsAfterID(id uint64) []*types.Alarm {
	var alarms types.Alarms
	ac.lock.RLock()
	for elem := ac.alarmList.Back(); elem != nil; elem = elem.Prev() {
		if elem.Value.(*types.Alarm).UID > id {
			alarms = append(alarms, elem.Value.(*types.Alarm))
		} else {
			break
		}
	}
	ac.lock.RUnlock()
	sort.Sort(alarms)
	return alarms
}

func (ac *AlarmCache) Add(alarm *types.Alarm) {
	if slice.SliceIndex(ClusterKinds, alarm.Kind) >= 0 {
		alarm.Namespace = ""
	}
	ac.lock.Lock()
	if ac.alarmList.Len() > 0 && isRepeat(ac.alarmList.Back().Value.(*types.Alarm), alarm) {
		return
	}

	alarm.UID = ac.eventID + 1
	alarm.SetID(uintToStr(alarm.UID))
	if err := addOrUpdateAlarmToDB(ac.alarmsTable, alarm, "add"); err != nil {
		log.Warnf("add alarm id [%s] to table failed: %s", *alarm, err)
		return
	}
	ac.alarmList.PushBack(alarm)
	ac.SetUnAck(1)
	if ac.alarmList.Len() > MaxAlarmCount {
		if err := ac.deleteoldest(); err != nil {
			log.Warnf("delete oldest alarms failed: %s", err)
		}
	}
	ac.lock.Unlock()

	atomic.AddUint64(&ac.eventID, 1)
	ac.cond.Broadcast()
	if err := SendMail(alarm, ac.thresholdTable); err != nil {
		log.Warnf("send mail failed: %s", err)
	}
}

func (ac *AlarmCache) Update(alarm *types.Alarm) error {
	if err := addOrUpdateAlarmToDB(ac.alarmsTable, alarm, "update"); err != nil {
		return err
	}
	ac.lock.Lock()
	a := ac.getAlarmFromList(alarm.UID)
	if a == nil {
		return fmt.Errorf("can not find alarm id %d", alarm.UID)
	}
	a.Acknowledged = true
	ac.lock.Unlock()
	ac.SetUnAck(-1)
	ac.cond.Broadcast()
	return nil
}

func (ac *AlarmCache) getAlarmFromList(id uint64) *types.Alarm {
	begID := ac.alarmList.Front().Value.(*types.Alarm).UID
	endID := ac.alarmList.Back().Value.(*types.Alarm).UID
	if id-begID < endID-id {
		return ac.getAlarmFromOutdated(id)
	} else {
		return ac.getAlarmFromLatest(id)
	}
}

func (ac *AlarmCache) getAlarmFromOutdated(id uint64) *types.Alarm {
	for elem := ac.alarmList.Front(); elem != nil; elem = elem.Next() {
		if elem.Value.(*types.Alarm).UID == id {
			return elem.Value.(*types.Alarm)
		}
	}
	return nil
}

func (ac *AlarmCache) getAlarmFromLatest(id uint64) *types.Alarm {
	for elem := ac.alarmList.Back(); elem != nil; elem = elem.Prev() {
		if elem.Value.(*types.Alarm).UID == id {
			return elem.Value.(*types.Alarm)
		}
	}
	return nil
}

func (ac *AlarmCache) deleteoldest() error {
	oneMonthLater := time.Now().AddDate(0, -1, 0)
	delNum := 1
	for elem := ac.alarmList.Front().Next(); elem != nil; elem = elem.Next() {
		if oneMonthLater.Before(time.Time(elem.Value.(*types.Alarm).Time)) {
			break
		}
		delNum += 1
	}
	firstID := ac.alarmList.Front().Value.(*types.Alarm).UID
	for i := 0; i < delNum; i++ {
		if err := deleteAlarmFromDB(ac.alarmsTable, uintToStr(uint64(i)+firstID)); err != nil {
			return err
		}
		elem := ac.alarmList.Front()
		ac.alarmList.Remove(elem)
		if !elem.Value.(*types.Alarm).Acknowledged {
			ac.SetUnAck(-1)
		}
	}
	return nil
}

func (ac *AlarmCache) deleteAlarmForCluster(cluster string) {
	ac.lock.Lock()
	var next *list.Element
	for elem := ac.alarmList.Front(); elem != nil; elem = next {
		next = elem.Next()
		alarm := elem.Value.(*types.Alarm)
		if alarm.Cluster == cluster {
			if err := deleteAlarmFromDB(ac.alarmsTable, uintToStr(alarm.UID)); err != nil {
				log.Warnf("delete alarm %d for cluster %s failed:%s", alarm.UID, cluster, err.Error())
				continue
			}
			ac.alarmList.Remove(elem)
			if !alarm.Acknowledged {
				ac.SetUnAck(-1)
			}
		}
	}
	ac.lock.Unlock()
}

func (ac *AlarmCache) SetUnAck(u int) {
	atomic.AddUint64(&ac.unAckNumber, uint64(u))
}

func uintToStr(uid uint64) string {
	return strconv.FormatInt(int64(uid), 10)
}

func isRepeat(lastAlarm, newAlarm *types.Alarm) bool {
	return lastAlarm.Cluster == newAlarm.Cluster &&
		lastAlarm.Namespace == newAlarm.Namespace &&
		lastAlarm.Kind == newAlarm.Kind &&
		lastAlarm.Reason == newAlarm.Reason &&
		lastAlarm.Message == newAlarm.Message &&
		lastAlarm.Name == newAlarm.Name
}
