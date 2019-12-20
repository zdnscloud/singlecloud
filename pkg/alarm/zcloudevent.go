package alarm

import (
	"fmt"
	"time"
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
		case ZcloudEvent:
			e := event.(ZcloudEvent)
			aw.Add(zcloudEventToAlarm(e))
		default:
		}
	}
}

func zcloudEventToAlarm(event ZcloudEvent) *Alarm {
	t := time.Now()
	return &Alarm{
		Time:      fmt.Sprintf("%.2d:%.2d:%.2d", t.Hour(), t.Minute(), t.Second()),
		Type:      ZcloudType,
		Namespace: event.Namespace,
		Kind:      event.Kind,
		Name:      event.Name,
		Reason:    event.Reason,
		Message:   event.Message,
	}
}
