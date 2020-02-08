package alarm

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

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
	firstID        uint64
	eventID        uint64
	maxSize        uint64
	unAckNumber    uint64
	alarms         map[string]*types.Alarm
}

type AlarmListener struct {
	lastID  uint64
	stopCh  chan struct{}
	alarmCh chan interface{}
}

func NewAlarmCache(size uint64) (*AlarmCache, error) {
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
		maxSize:        size,
		stopCh:         stop,
		thresholdTable: thresholdTable,
		alarmsTable:    alarmsTable,
		alarms:         make(map[string]*types.Alarm),
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
	if len(alarms) == 0 {
		ac.firstID = 1
		return nil
	}
	ac.firstID = alarms[0].UID
	for _, alarm := range alarms {
		if alarm.UID < ac.firstID {
			ac.firstID = alarm.UID
		}
		if alarm.UID >= ac.eventID {
			ac.eventID = alarm.UID
		}
		if !alarm.Acknowledged {
			ac.unAckNumber += 1
		}
		ac.alarms[uintToStr(alarm.UID)] = alarm
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
			case al.alarmCh <- *alarm:
			}
		}
	}
}

func (ac *AlarmCache) getAlarmsAfterID(id uint64) []*types.Alarm {
	ac.lock.RLock()
	defer ac.lock.RUnlock()
	var res types.Alarms
	for _, alarm := range ac.alarms {
		if !alarm.Acknowledged && alarm.UID > id {
			res = append(res, alarm)
		}
	}
	sort.Sort(res)
	return res
}

func (ac *AlarmCache) Add(alarm *types.Alarm) {
	if slice.SliceIndex(ClusterKinds, alarm.Kind) >= 0 {
		alarm.Namespace = ""
	}
	if len(ac.alarms) > 0 && isRepeat(ac.alarms[uintToStr(ac.eventID)], alarm) {
		return
	}

	alarm.UID = ac.eventID + 1
	alarm.SetID(strconv.Itoa(int(alarm.UID)))
	if err := addOrUpdateAlarmToDB(ac.alarmsTable, alarm, "add"); err != nil {
		log.Warnf("add alarm id %d to table failed: %s", alarm.UID, err)
		return
	}
	ac.alarms[uintToStr(alarm.UID)] = alarm
	if err := SendMail(alarm, ac.thresholdTable); err != nil {
		log.Warnf("send mail failed: %s", err)
	}

	ackNum := 1
	if uint64(len(ac.alarms)) > ac.maxSize {
		del, err := ac.delOver()
		if err != nil {
			log.Warnf("delete the alarm out of queue failed: %s", err)
		}
		ackNum = ackNum - del
	}

	ac.eventIDAtomicAdd()
	ac.SetUnAck(ackNum)
}

func isRepeat(lastAlarm, newAlarm *types.Alarm) bool {
	return lastAlarm.Cluster == newAlarm.Cluster &&
		lastAlarm.Namespace == newAlarm.Namespace &&
		lastAlarm.Kind == newAlarm.Kind &&
		lastAlarm.Reason == newAlarm.Reason &&
		lastAlarm.Message == newAlarm.Message &&
		lastAlarm.Name == newAlarm.Name
}

func (ac *AlarmCache) Update(alarm *types.Alarm) error {
	if err := addOrUpdateAlarmToDB(ac.alarmsTable, alarm, "update"); err != nil {
		return err
	}
	ac.alarms[uintToStr(alarm.UID)].Acknowledged = true
	ac.SetUnAck(-1)
	return nil
}

func (ac *AlarmCache) Del(id uint64) error {
	if err := deleteAlarmFromDB(ac.alarmsTable, uintToStr(id)); err != nil {
		return err
	}
	delete(ac.alarms, uintToStr(id))
	ac.firstID += 1
	return nil
}

func (ac *AlarmCache) eventIDAtomicAdd() {
	ac.lock.Lock()
	defer ac.lock.Unlock()
	atomic.AddUint64(&ac.eventID, 1)
}

func (ac *AlarmCache) SetUnAck(u int) {
	ac.lock.Lock()
	defer ac.lock.Unlock()
	atomic.AddUint64(&ac.unAckNumber, uint64(u))
	ac.cond.Broadcast()
}

func uintToStr(uid uint64) string {
	return strconv.FormatInt(int64(uid), 10)
}
