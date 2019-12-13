package alarm

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/zdnscloud/cement/uuid"
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
		case Alarm:
			alarm := event.(Alarm)
			uid, _ := uuid.Gen()
			t := time.Now()
			alarm.ID = atomic.AddUint64(&aw.eventID, 1)
			alarm.UUID = uid
			alarm.Time = fmt.Sprintf("%.2d:%.2d:%.2d", t.Hour(), t.Minute(), t.Second())
			aw.Add(&alarm)
		default:
		}
	}
}
