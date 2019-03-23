package handler

import (
	"bufio"
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/singlecloud/hack/sockjs"
	"github.com/zdnscloud/singlecloud/pkg/logger"
)

var (
	MaxLineCountFromTail = int64(1000)
	LogRequestTimeout    = 10 * time.Second
)

func (m *ClusterManager) OpenPodLog(clusterID, namespace, pod, container string, r *http.Request, w http.ResponseWriter) {
	cluster := m.get(clusterID)
	if cluster == nil {
		logger.Warn("cluster %s isn't found to open console", clusterID)
		return
	}

	Sockjshandler := func(session sockjs.Session) {
		podClient, _ := cluster.KubeClient.RestClientForObject(&corev1.Pod{}, LogRequestTimeout)
		opts := corev1.PodLogOptions{
			Follow:     true,
			Container:  container,
			Timestamps: true,
			TailLines:  &MaxLineCountFromTail,
		}
		req := podClient.
			Get().
			Namespace(namespace).
			Name(pod).
			Resource("pods").
			SubResource("log").
			VersionedParams(&opts, scheme.ParameterCodec)
		readCloser, err := req.Stream()
		if err != nil {
			logger.Warn("request log get err %s", err.Error())
			session.Close(503, err.Error())
			return
		}

		done := make(chan struct{})
		go func() {
			<-session.ClosedNotify()
			readCloser.Close()
			close(done)
		}()

		s := bufio.NewScanner(readCloser)
		for s.Scan() {
			err := session.Send(string(s.Bytes()))
			if err != nil {
				logger.Warn("send log failed:%s", err.Error())
				break
			}
		}
		session.Close(503, "log is terminated")
		<-done
	}

	path := fmt.Sprintf(WSPodLogPathTemp, clusterID, namespace, pod, container)
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
