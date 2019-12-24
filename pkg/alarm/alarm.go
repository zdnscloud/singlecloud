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
	ID           uint64    `json:"-"`
	Time         string    `json:"time,omitempty"`
	Cluster      string    `json:"cluster,omitempty"`
	Type         AlarmType `json:"type,omitempty"`
	Namespace    string    `json:"namespace,omitempty"`
	Kind         string    `json:"kind,omitempty"`
	Name         string    `json:"name,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Message      string    `json:"message,omitempty"`
	Acknowledged bool      `json:"acknowledged,omitempty"`
}

type AlarmEvent struct {
	Alarm
}

func New() *AlarmEvent {
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
