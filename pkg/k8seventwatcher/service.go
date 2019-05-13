package k8seventwatcher

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/hack/sockjs"
)

const (
	WSPrefix        = "/apis/ws.zcloud.cn/v1"
	WSEventPathTemp = WSPrefix + "/clusters/%s/event"
)

func (mgr *WatcherManager) RegisterHandler(router gin.IRoutes) error {
	eventPath := fmt.Sprintf(WSEventPathTemp, ":cluster") + "/*actions"
	router.GET(eventPath, func(c *gin.Context) {
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

	Sockjshandler := func(session sockjs.Session) {
		listener := watcher.AddListener()
		done := make(chan struct{})
		go func() {
			<-session.ClosedNotify()
			listener.Stop()
			close(done)
		}()

		eventCh := listener.EventChannel()
		for {
			e, ok := <-eventCh
			if ok == false {
				break
			}

			event := map[string]string{
				"id":        string(e.UID),
				"time":      e.CreationTimestamp.Format("3:04PM"),
				"namespace": e.Namespace,
				"type":      e.Type,
				"kind":      e.InvolvedObject.Kind,
				"name":      e.InvolvedObject.Name,
				"reason":    e.Reason,
				"message":   e.Message,
				"source":    e.Source.Component + "," + e.Source.Host,
			}
			d, _ := json.Marshal(event)
			err := session.Send(string(d))
			if err != nil {
				log.Warnf("send log failed:%s", err.Error())
				break
			}
		}
		session.Close(503, "event is terminated")
		<-done
	}

	path := fmt.Sprintf(WSEventPathTemp, clusterID)
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
