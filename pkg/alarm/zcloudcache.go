package alarm

import (
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
)

func publishAlarmEvent(aw *AlarmCache, stop chan struct{}) {
	alarmEventCh := eventBus.Sub(eventbus.AlarmEvent)
	for {
		select {
		case <-stop:
			return
		default:
		}
		event := <-alarmEventCh
		switch event.(type) {
		case *Alarm:
			alarm := event.(*Alarm)
			alarm.Type = ZcloudType
			aw.Add(alarm)
		default:
		}
	}
}
