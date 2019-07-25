package handler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/hack/sockjs"
)

var (
	MaxLineCountFromTail = int64(1000)
	LogRequestTimeout    = 5 * time.Second
)

func (m *ClusterManager) openPodLog(cluster *Cluster, namespace, pod, container string) (io.ReadCloser, error) {
	//when container has no log, Stream call will block forever
	//if set client timeout, Stream will be timed out too
	//so check whether there is any log first
	oneline := int64(1)
	podClient, _ := cluster.KubeClient.RestClientForObject(&corev1.Pod{}, LogRequestTimeout)
	opts := corev1.PodLogOptions{
		Follow:     false,
		Container:  container,
		Timestamps: false,
		TailLines:  &oneline,
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
		return nil, err
	}
	buf := make([]byte, 8)
	n, err := io.ReadFull(readCloser, buf)
	readCloser.Close()
	if n == 0 || err != nil {
		return nil, io.EOF
	}

	podClient, _ = cluster.KubeClient.RestClientForObject(&corev1.Pod{}, 0)
	opts = corev1.PodLogOptions{
		Follow:     true,
		Container:  container,
		Timestamps: true,
		TailLines:  &MaxLineCountFromTail,
	}
	req = podClient.
		Get().
		Namespace(namespace).
		Name(pod).
		Resource("pods").
		SubResource("log").
		VersionedParams(&opts, scheme.ParameterCodec)
	return req.Stream()
}

func (m *ClusterManager) OpenPodLog(clusterID, namespace, pod, container string, r *http.Request, w http.ResponseWriter) {
	cluster := m.getReady(clusterID)
	if cluster == nil {
		log.Warnf("cluster %s isn't found to open console", clusterID)
		return
	}

	Sockjshandler := func(session sockjs.Session) {
		readCloser, err := m.openPodLog(cluster, namespace, pod, container)
		if err != nil {
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
				log.Warnf("send log failed:%s", err.Error())
				break
			}
		}
		session.Close(503, "log is terminated")
		<-done
	}

	path := fmt.Sprintf(WSPodLogPathTemp, clusterID, namespace, pod, container)
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
