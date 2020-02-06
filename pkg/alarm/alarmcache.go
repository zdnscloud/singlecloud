package alarm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/db"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	AlarmTable = "alarm"
)

type AlarmCache struct {
	eventID        uint64
	maxSize        uint
	lock           sync.RWMutex
	cond           *sync.Cond
	stopCh         chan struct{}
	unAckNumber    uint64
	ackCh          chan int
	ThresholdTable kvzoo.Table
	alarmTable     kvzoo.Table
}

type AlarmListener struct {
	lastID  uint64
	stopCh  chan struct{}
	alarmCh chan interface{}
}

func NewAlarmCache(size uint) (*AlarmCache, error) {
	thresholdTable, err := genTable(types.ThresholdTable)
	if err != nil {
		return nil, err
	}
	alarmTable, err := genTable(AlarmTable)
	if err != nil {
		return nil, err
	}
	stop := make(chan struct{})
	ac := &AlarmCache{
		maxSize:        size,
		stopCh:         stop,
		ackCh:          make(chan int),
		ThresholdTable: thresholdTable,
		alarmTable:     alarmTable,
	}
	alarms, err := getAlarmsFromDB(alarmTable)
	if err != nil {
		return nil, err
	}
	if len(alarms) > 0 {
		ac.eventID = sortAlarms(alarms)[0].UID
	}
	ac.unAckNumber = getUnackAlarmsNumber(alarms)
	ac.cond = sync.NewCond(&ac.lock)
	go subscribeAlarmEvent(ac, stop)
	return ac, nil
}

func getAlarmsFromDB(table kvzoo.Table) ([]*types.Alarm, error) {
	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Commit()
	values, err := tx.List()
	if err != nil {
		return nil, err
	}
	var alarms types.Alarms
	for _, value := range values {
		var alarm types.Alarm
		if err := json.Unmarshal(value, &alarm); err != nil {
			return nil, err
		}
		alarms = append(alarms, &alarm)
	}
	sort.Sort(alarms)
	return alarms, nil
}

func deleteAlarmFromDB(table kvzoo.Table, id string) error {
	tx, err := table.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := tx.Delete(id); err != nil {
		return err
	}

	return tx.Commit()
}

func addOrUpdateAlarmToDB(table kvzoo.Table, alarm *types.Alarm, action string) error {
	value, err := json.Marshal(alarm)
	if err != nil {
		return fmt.Errorf("marshal list %s failed: %s", alarm.UID, err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %s", err.Error())
	}

	defer tx.Rollback()
	switch action {
	case "add":
		if err = tx.Add(strconv.FormatInt(int64(alarm.UID), 10), value); err != nil {
			return err
		}
	case "update":
		if err = tx.Update(strconv.FormatInt(int64(alarm.UID), 10), value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func genTable(name string) (kvzoo.Table, error) {
	tn, _ := kvzoo.TableNameFromSegments(name)
	table, err := db.GetGlobalDB().CreateOrGetTable(tn)
	if err != nil {
		return nil, fmt.Errorf("create or get table %s failed: %s", name, err.Error())
	}
	return table, nil
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
		alarms, err := ac.getAlarmsAfterID(al.lastID)
		if err != nil {
			continue
		}
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

func (ac *AlarmCache) getAlarmsAfterID(id uint64) ([]*types.Alarm, error) {
	ac.lock.RLock()
	defer ac.lock.RUnlock()
	var res types.Alarms
	alarms, err := getAlarmsFromDB(ac.alarmTable)
	if err != nil {
		return nil, err
	}
	for _, alarm := range alarms {
		if !alarm.Acknowledged && alarm.UID > id {
			res = append(res, alarm)
		}
	}
	sort.Sort(res)
	return res, nil
}

func (ac *AlarmCache) Add(alarm *types.Alarm) {
	ac.lock.Lock()
	defer ac.lock.Unlock()
	if slice.SliceIndex(ClusterKinds, alarm.Kind) >= 0 {
		alarm.Namespace = ""
	}
	alarms, err := getAlarmsFromDB(ac.alarmTable)
	if err != nil {
		log.Warnf("get alarms from db failed: %s", err)
		return
	}
	if len(alarms) > 0 && isRepeat(alarms[0], alarm) {
		return
	}

	alarm.UID = atomic.AddUint64(&ac.eventID, 1)
	alarm.SetID(strconv.Itoa(int(alarm.UID)))
	if err := addOrUpdateAlarmToDB(ac.alarmTable, alarm, "add"); err != nil {
		log.Warnf("add alarm id %d to table failed: %s", alarm.UID, err)
		return
	}
	if err := SendMail(alarm, ac.ThresholdTable); err != nil {
		log.Warnf("send mail failed: %s", err)
	}
	ackNum := 1
	for uint(len(alarms)+1) > ac.maxSize {
		delAlarm := alarms[(len(alarms) - 1)]
		if !delAlarm.Acknowledged {
			ackNum -= 1
		}
		if err := deleteAlarmFromDB(ac.alarmTable, strconv.FormatInt(int64(delAlarm.UID), 10)); err != nil {
			log.Warnf("delete alarm id %d from table failed: %s", alarm.UID, err)
		}
		alarms = append(alarms[:(len(alarms)-1)], alarms[len(alarms):]...)
	}
	ac.SetUnAck(ackNum)

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

func sortAlarms(alarms []*types.Alarm) []*types.Alarm {
	var res types.Alarms
	res = alarms
	sort.Sort(res)
	return res
}

func getUnackAlarmsNumber(alarms []*types.Alarm) uint64 {
	var i uint64
	for _, alarm := range alarms {
		if !alarm.Acknowledged {
			i += 1
		}
	}
	return i
}
