package alarm

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	WSPrefix            = "/apis/ws.zcloud.cn/v1"
	eventPath           = WSPrefix + "/alarm"
	alarmUpdateLink     = "/apis/zcloud.cn/v1/alarms/%s"
	alarmCollectionLink = "/apis/zcloud.cn/v1/alarms"
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

	listener := mgr.cache.AddListener()
	alarmCh := listener.AlarmChannel()
	for {
		alarm, ok := <-alarmCh
		if ok == false {
			break
		}
		var msg Message
		switch alarm.(type) {
		case *types.Alarm:
			a := alarm.(*types.Alarm)
			genLink(a)
			msg.Type = UnackAlarm
			msg.Payload = a
		case int:
			msg.Type = UnackNumber
			msg.Payload = alarm.(int)
		}
		err = conn.WriteJSON(msg)
		if err != nil {
			log.Warnf("send alarm failed:%s", err.Error())
			break
		}
	}
}

func genLink(alarm *types.Alarm) {
	links := make(map[resource.ResourceLinkType]resource.ResourceLink)
	links[resource.CollectionLink] = alarmCollectionLink
	links[resource.UpdateLink] = resource.ResourceLink(fmt.Sprintf(alarmUpdateLink, alarm.GetID()))
	alarm.SetLinks(links)
}
