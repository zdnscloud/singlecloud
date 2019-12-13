package k8seventwatcher

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/uuid"
)

const (
	WSPrefix        = "/apis/ws.zcloud.cn/v1"
	WSEventPathTemp = WSPrefix + "/clusters/%s/event"
)

func (mgr *WatcherManager) RegisterHandler(router gin.IRoutes) error {
	eventPath := fmt.Sprintf(WSEventPathTemp, ":cluster")
	router.GET(eventPath, func(c *gin.Context) {
		fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!")
		fmt.Println(c.Request.Context().Value(types.CurrentUserKey))
		mgr.OpenEvent(c.Param("cluster"), c.Request, c.Writer)
	})

	return nil
}

func (mgr *WatcherManager) OpenEvent(clusterID string, r *http.Request, w http.ResponseWriter) {
	mgr.lock.Lock()
	watcher, ok := mgr.watchers[clusterID]
	mgr.lock.Unlock()

	if ok == false {
		log.Warnf("cluster %s isn't found to open console", clusterID)
		return
	}

	conn, err := websocket.Upgrade(w, r, nil, 4096, 4096)
	if err != nil {
		log.Warnf("cluster %s event websocket upgrade failed %s", clusterID, err.Error())
		return
	}
	defer conn.Close()

	listener := watcher.AddListener()

	eventCh := listener.EventChannel()
	for {
		e, ok := <-eventCh
		if ok == false {
			break
		}
		//event id in k8s may duplicate, generate uuid by ourselve
		id, _ := uuid.Gen()
		t := e.LastTimestamp
		event := map[string]string{
			"id":        id,
			"time":      fmt.Sprintf("%.2d:%.2d:%.2d", t.Hour(), t.Minute(), t.Second()),
			"namespace": e.Namespace,
			"type":      e.Type,
			"kind":      e.InvolvedObject.Kind,
			"name":      e.InvolvedObject.Name,
			"reason":    e.Reason,
			"message":   e.Message,
			"source":    e.Source.Component + "," + e.Source.Host,
		}
		err := conn.WriteJSON(event)
		if err != nil {
			log.Warnf("send log failed:%s", err.Error())
			break
		}
	}
}
