package zke

import (
	"fmt"
	"net/http"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/hack/sockjs"
)

const (
	MaxZKELogLines   = 100
	WSPrefix         = "/apis/ws.zcloud.cn/v1"
	WSZKELogPathTemp = WSPrefix + "/clusters/%s/zkelog"
)

func (m ZKEManager) OpenLog(id string, r *http.Request, w http.ResponseWriter) {
	cluster, ok := m[id]
	if !ok {
		log.Warnf("cluster %s isn't found to open log console", id)
		return
	}

	if cluster.logCh == nil {
		log.Warnf("cluster log channel is nil can't to open log console", id)
		return
	}

	if cluster.logSession != nil {
		cluster.lock.Lock()
		cluster.logSession.Close(503, "new connection is opened")
		cluster.lock.Unlock()
	}

	Sockjshandler := func(session sockjs.Session) {
		done := make(chan struct{})
		cluster.lock.Lock()
		cluster.logSession = session
		cluster.lock.Unlock()
		go func() {
			<-session.ClosedNotify()
			close(done)
		}()

		for {
			logString, ok := <-cluster.logCh
			if !ok {
				break
			}
			err := session.Send(logString)
			if err != nil {
				log.Warnf("send log failed:%s", err.Error())
				break
			}
		}
		cluster.lock.Lock()
		session.Close(503, "log is terminated")
		cluster.logSession = nil
		cluster.lock.Unlock()
		<-done
	}

	path := fmt.Sprintf(WSZKELogPathTemp, id)
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
