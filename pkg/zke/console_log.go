package zke

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
)

const (
	MaxZKELogLines   = 50
	WSPrefix         = "/apis/ws.zcloud.cn/v1"
	WSZKELogPathTemp = WSPrefix + "/clusters/%s/zkelog"
)

func (m *ZKEManager) OpenLog(clusterID string, r *http.Request, w http.ResponseWriter) {
	cluster := m.get(clusterID)
	if cluster == nil {
		log.Warnf("cluster %s isn't found to open log console", clusterID)
		return
	}

	if cluster.logCh == nil {
		log.Warnf("cluster %s log channel is empty", clusterID)
		return
	}
	cluster.openLog(r, w)
}

func (c *Cluster) openLog(r *http.Request, w http.ResponseWriter) {
	if c.logSession != nil {
		c.lock.Lock()
		c.logSession.Close()
		c.lock.Unlock()
	}

	conn, err := websocket.Upgrade(w, r, nil, 4096, 4096)
	if err != nil {
		log.Warnf("cluster %s console log websocket upgrade failed %s", c.Name, err.Error())
	}
	defer conn.Close()

	c.lock.Lock()
	c.logSession = conn
	c.lock.Unlock()

	for {
		logString, ok := <-c.logCh
		if !ok {
			break
		}

		err := conn.WriteMessage(websocket.TextMessage, []byte(logString))
		if err != nil {
			log.Warnf("send cluster %s console log failed:%s", c.Name, err.Error())
			break
		}
	}
}
