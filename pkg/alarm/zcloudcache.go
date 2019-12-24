package alarm

import (
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
)

func publishAlarmEvent(ac *AlarmCache, stop chan struct{}) {
	alarmEventCh := eventBus.Sub(eventbus.AlarmEvent)
	for {
		select {
		case <-stop:
			return
		case event := <-alarmEventCh:
			alarm := event.(*Alarm)
			alarm.Type = ZcloudType
			ac.Add(alarm)
		}
	}
}
