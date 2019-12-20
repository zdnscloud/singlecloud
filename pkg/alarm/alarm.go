package alarm

import (
	"fmt"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/eventbus"
)

const (
	EventType  AlarmType = "Event"
	ZcloudType AlarmType = "Alarm"
)

type AlarmType string

type Alarm struct {
	ID           uint64
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

type AlarmEvent struct {
	Alarm
}

func NewAlarm() *AlarmEvent {
	t := time.Now()
	time := fmt.Sprintf("%.2d:%.2d:%.2d", t.Hour(), t.Minute(), t.Second())
	return &AlarmEvent{Alarm{Time: time}}
}

func (a *AlarmEvent) Namespace(namespace string) *AlarmEvent {
	a.Alarm.Namespace = namespace
	return a
}

func (a *AlarmEvent) Kind(kind string) *AlarmEvent {
	a.Alarm.Kind = kind
	return a
}

func (a *AlarmEvent) Name(name string) *AlarmEvent {
	a.Alarm.Name = name
	return a
}

func (a *AlarmEvent) Message(message string) *AlarmEvent {
	a.Alarm.Message = message
	return a
}

func (a *AlarmEvent) Reason(reason string) *AlarmEvent {
	a.Alarm.Reason = reason
	return a
}

func (a *AlarmEvent) Publish() {
	eventBus.Pub(&a.Alarm, eventbus.AlarmEvent)
}
