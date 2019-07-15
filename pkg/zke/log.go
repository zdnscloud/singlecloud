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

func (z *ZKE) OpenLog(clusterID string, r *http.Request, w http.ResponseWriter) {
	cluster := z.Get(clusterID)
	if cluster == nil {
		log.Warnf("cluster %s isn't found to open log console", clusterID)
		return
	}

	if cluster.logCh == nil {
		log.Warnf("cluster log channel is nil can't to open log console", clusterID)
		return
	}

	if cluster.logSession != nil {
		z.Lock.Lock()
		cluster.logSession.Close(503, "new connection is opened")
		z.Lock.Unlock()
	}

	Sockjshandler := func(session sockjs.Session) {
		done := make(chan struct{})
		z.Lock.Lock()
		cluster.logSession = session
		z.Lock.Unlock()
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
		z.Lock.Lock()
		session.Close(503, "log is terminated")
		cluster.logSession = nil
		z.Lock.Unlock()
		<-done
	}

	path := fmt.Sprintf(WSZKELogPathTemp, clusterID)
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
