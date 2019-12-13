package alarm

import (
	"container/list"
	"sync"
)

const (
	EventType    AlarmType = "Event"
	ZcloudType   AlarmType = "Zcloud"
	ResourceType AlarmType = "Resource"
)

type AlarmType string

type AlarmWatcher struct {
	eventID       uint64
	maxSize       uint
	lock          sync.RWMutex
	cond          *sync.Cond
	alarmList     *list.List
	stopCh        chan struct{}
	unAckNumber   int
	ackCh         chan int
	zcloudEventCh <-chan interface{}
}

type AlarmListener struct {
	lastID  uint64
	stopCh  chan struct{}
	alarmCh chan Alarm
}

type Alarm struct {
	ID           uint64 `json:"-"`
	UUID         string
	Time         string
	Type         AlarmType
	Namespace    string
	Kind         string
	Name         string
	Reason       string
	Message      string
	Acknowledged bool
}
