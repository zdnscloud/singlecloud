package alarm

import (
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func subscribeAlarmEvent(cache *AlarmCache, stop chan struct{}) {
	alarmEventCh := eventBus.Sub(eventbus.AlarmEvent)
	for {
		select {
		case <-stop:
			return
		case event := <-alarmEventCh:
			alarm := event.(*types.Alarm)
			alarm.Type = types.ZcloudType
			cache.Add(alarm)
		}
	}
}
