package zke

import (
	"fmt"
	"net/http"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/hack/sockjs"
)

const (
	MaxZKELogLines   = 50
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
		log.Warnf("only cluster in createing and updateing state can open zke log console", id)
		return
	}
	cluster.openLog(id, r, w)
}

func (c *ZKECluster) openLog(id string, r *http.Request, w http.ResponseWriter) {
	if c.logSession != nil {
		c.lock.Lock()
		c.logSession.Close(503, "new connection is opened")
		c.lock.Unlock()
	}

	Sockjshandler := func(session sockjs.Session) {
		done := make(chan struct{})
		c.lock.Lock()
		c.logSession = session
		c.lock.Unlock()
		go func() {
			<-session.ClosedNotify()
			close(done)
		}()

		for {
			logString, ok := <-c.logCh
			if !ok {
				break
			}
			err := session.Send(logString)
			if err != nil {
				log.Warnf("send log failed:%s", err.Error())
				break
			}
		}
		c.lock.Lock()
		session.Close(503, "log is terminated")
		c.logSession = nil
		c.lock.Unlock()
		<-done
	}

	path := fmt.Sprintf(WSZKELogPathTemp, id)
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
