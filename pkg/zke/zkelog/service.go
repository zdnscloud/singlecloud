package zkelog

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
)

const (
	WSPrefix         = "/apis/ws.zcloud.cn/v1"
	WSZKELogPathTemp = WSPrefix + "/clusters/%s/zkelog"
)

func (m *LogManager) OpenLog(id string, r *http.Request, w http.ResponseWriter) {
	watcher := m.get(id)
	if watcher == nil {
		log.Warnf("cluster %s log watcher isn't found to open console", id)
		return
	}

	listener := watcher.addListener()
	defer listener.Stop()

	conn, err := websocket.Upgrade(w, r, nil, 4096, 4096)
	if err != nil {
		log.Warnf("cluster %s console log websocket upgrade failed %s", id, err.Error())
	}
	defer conn.Close()

	logCh := listener.LogChannel()

	for {
		logString, ok := <-logCh
		if !ok {
			break
		}

		if err := conn.WriteMessage(websocket.TextMessage, []byte(logString)); err != nil {
			if !isBrokenPipeErr(err) {
				log.Warnf("send cluster %s console log failed:%s", id, err.Error())
			}
			break
		}
	}
}

func isBrokenPipeErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "broken pipe") ||
		strings.Contains(strings.ToLower(err.Error()), "connection reset by peer")
}
