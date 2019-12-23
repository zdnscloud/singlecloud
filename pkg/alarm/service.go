package alarm

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
)

const (
	WSPrefix        = "/apis/ws.zcloud.cn/v1"
	WSEventPathTemp = WSPrefix + "/clusters/%s/alarm"
	WSUnAckPathTemp = WSPrefix + "/clusters/%s/unack"
)

func (mgr *AlarmManager) RegisterHandler(router gin.IRoutes) error {
	eventPath := fmt.Sprintf(WSEventPathTemp, ":cluster")
	router.GET(eventPath, func(c *gin.Context) {
		mgr.OpenAlarm(c.Param("cluster"), c.Request, c.Writer)
	})

	unAckPath := fmt.Sprintf(WSUnAckPathTemp, ":cluster")
	router.GET(unAckPath, func(c *gin.Context) {
		mgr.OpenUnAck(c.Param("cluster"), c.Request, c.Writer)
	})

	return nil
}

func (mgr *AlarmManager) OpenUnAck(clusterID string, r *http.Request, w http.ResponseWriter) {
	mgr.lock.Lock()
	cache, ok := mgr.caches[clusterID]
	mgr.lock.Unlock()

	if ok == false {
		log.Warnf("cluster %s isn't found to open console", clusterID)
		return
	}

	conn, err := websocket.Upgrade(w, r, nil, 0, 0)
	if err != nil {
		log.Warnf("cluster %s event websocket upgrade failed %s", clusterID, err.Error())
		return
	}
	defer conn.Close()

	ackCh := cache.AckChannel()
	for {
		err := conn.WriteJSON(cache.unAckNumber)
		if err != nil {
			log.Warnf("send log failed:%s", err.Error())
			break
		}
		_, ok := <-ackCh
		if ok == false {
			break
		}
	}
}

func (mgr *AlarmManager) OpenAlarm(clusterID string, r *http.Request, w http.ResponseWriter) {
	mgr.lock.Lock()
	cache, ok := mgr.caches[clusterID]
	mgr.lock.Unlock()

	if ok == false {
		log.Warnf("cluster %s isn't found to open console", clusterID)
		return
	}

	conn, err := websocket.Upgrade(w, r, nil, 0, 0)
	if err != nil {
		log.Warnf("cluster %s event websocket upgrade failed %s", clusterID, err.Error())
		return
	}
	defer conn.Close()

	listener := cache.AddListener()

	alarmCh := listener.AlarmChannel()
	for {
		e, ok := <-alarmCh
		if ok == false {
			break
		}
		err = conn.WriteJSON(e)
		if err != nil {
			log.Warnf("send log failed:%s", err.Error())
			break
		}
	}
}
