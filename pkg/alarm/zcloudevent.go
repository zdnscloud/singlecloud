package alarm

import (
	"fmt"
	"time"

	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
)

func publishZloudEvent(aw *AlarmWatcher, stop chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
		}
		event := <-aw.zcloudEventCh
		switch event.(type) {
		case *ZcloudAlarm:
			e := event.(*ZcloudAlarm)
			aw.Add(zcloudAlarmToAlarm(e))
		default:
		}
	}
}

func zcloudAlarmToAlarm(event *ZcloudAlarm) *Alarm {
	t := time.Now()
	return &Alarm{
		Time:      fmt.Sprintf("%.2d:%.2d:%.2d", t.Hour(), t.Minute(), t.Second()),
		Type:      ZcloudType,
		Namespace: event.namespace,
		Kind:      event.kind,
		Name:      event.name,
		Reason:    event.reason,
		Message:   event.message,
	}
}

func NewZcloudAlarm(eventBus *pubsub.PubSub) *ZcloudAlarm {
	alarm := &ZcloudAlarm{}
	alarm.eventBus = eventBus
	return alarm
}

func (a *ZcloudAlarm) Namespace(namespace string) *ZcloudAlarm {
	a.namespace = namespace
	return a
}

func (a *ZcloudAlarm) Kind(kind string) *ZcloudAlarm {
	a.kind = kind
	return a
}

func (a *ZcloudAlarm) Name(name string) *ZcloudAlarm {
	a.name = name
	return a
}

func (a *ZcloudAlarm) Message(message string) *ZcloudAlarm {
	a.message = message
	return a
}

func (a *ZcloudAlarm) Reason(reason string) *ZcloudAlarm {
	a.reason = reason
	return a
}

func (a *ZcloudAlarm) Pub() {
	a.eventBus.Pub(a, eventbus.AlarmEvent)
}
