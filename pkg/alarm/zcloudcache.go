package alarm

import (
	eb "github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func subscribeAlarmEvent(cache *AlarmCache, stop chan struct{}) {
	alarmEventCh := eb.SubscribeResourceEvent(types.Alarm{})
	for {
		select {
		case <-stop:
			return
		case event := <-alarmEventCh:
			switch e := event.(type) {
			case eb.ResourceCreateEvent:
				alarm := e.Resource.(*types.Alarm)
				alarm.Type = types.ZcloudType
				cache.Add(alarm)
			}

		}
	}
}
