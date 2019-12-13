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

func (mgr *WatcherManager) RegisterHandler(router gin.IRoutes) error {
	eventPath := fmt.Sprintf(WSEventPathTemp, ":cluster")
	router.GET(eventPath, func(c *gin.Context) {
		mgr.OpenEvent(c.Param("cluster"), c.Request, c.Writer)
	})

	unAckPath := fmt.Sprintf(WSUnAckPathTemp, ":cluster")
	router.GET(unAckPath, func(c *gin.Context) {
		mgr.OpenUnAck(c.Param("cluster"), c.Request, c.Writer)
	})

	return nil
}

func (mgr *WatcherManager) OpenUnAck(clusterID string, r *http.Request, w http.ResponseWriter) {
	mgr.lock.Lock()
	watcher, ok := mgr.watchers[clusterID]
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

	ackCh := watcher.AckChannel()
	for {
		err := conn.WriteJSON(watcher.unAckNumber)
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

func (mgr *WatcherManager) OpenEvent(clusterID string, r *http.Request, w http.ResponseWriter) {
	mgr.lock.Lock()
	watcher, ok := mgr.watchers[clusterID]
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

	listener := watcher.AddListener()

	alarmCh := listener.AlarmChannel()
	for {
		e, ok := <-alarmCh
		if ok == false {
			break
		}
		fmt.Println("=====in gin", e)
		//event id in k8s may duplicate, generate uuid by ourselve
		err := conn.WriteJSON(e)
		if err != nil {
			log.Warnf("send log failed:%s", err.Error())
			break
		}
	}
}
