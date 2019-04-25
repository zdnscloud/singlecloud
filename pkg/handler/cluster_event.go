package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/hack/sockjs"
)

func (m *ClusterManager) OpenEvent(clusterID string, r *http.Request, w http.ResponseWriter) {
	cluster := m.get(clusterID)
	if cluster == nil {
		log.Warnf("cluster %s isn't found to open console", clusterID)
		return
	}

	Sockjshandler := func(session sockjs.Session) {
		listener := cluster.EventWatcher.AddListener()

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
				"time":      e.FirstTimestamp.Format("3:04PM"),
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
