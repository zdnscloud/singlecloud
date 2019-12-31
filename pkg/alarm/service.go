package alarm

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	WSPrefix  = "/apis/ws.zcloud.cn/v1"
	eventPath = WSPrefix + "/alarm"
)

func (mgr *AlarmManager) RegisterHandler(router gin.IRoutes) error {
	router.GET(eventPath, func(c *gin.Context) {
		mgr.OpenAlarm(c.Request, c.Writer)
	})
	return nil
}

func (mgr *AlarmManager) OpenAlarm(r *http.Request, w http.ResponseWriter) {
	conn, err := websocket.Upgrade(w, r, nil, 0, 0)
	if err != nil {
		log.Warnf("event websocket upgrade failed %s", err.Error())
		return
	}
	defer conn.Close()

	err = conn.WriteJSON(Message{UnackNumber, mgr.cache.unAckNumber})
	if err != nil {
		log.Warnf("send log failed:%s", err.Error())
	}

	listener := mgr.cache.AddListener()
	alarmCh := listener.AlarmChannel()
	for {
		alarm, ok := <-alarmCh
		if ok == false {
			break
		}
		var msg Message
		switch alarm.(type) {
		case types.Alarm:
			msg.Type = UnackAlarm
			msg.Payload = alarm.(types.Alarm)
		case uint64:
			msg.Type = UnackNumber
			msg.Payload = alarm.(uint64)
		}
		err = conn.WriteJSON(msg)
		if err != nil {
			log.Warnf("send log failed:%s", err.Error())
			break
		}
	}
}
