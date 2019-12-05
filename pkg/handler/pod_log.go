package handler

import (
	"bufio"
	"io"
	"net/http"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/zke"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
)

var (
	MaxLineCountFromTail = int64(1000)
	LogRequestTimeout    = 5 * time.Second
)

func (m *ClusterManager) openPodLog(cluster *zke.Cluster, namespace, pod, container string) (io.ReadCloser, error) {
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
	cluster := m.GetClusterByName(clusterID)
	if cluster == nil {
		log.Warnf("cluster %s isn't found to open console", clusterID)
		return
	}

	var upgrader = websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("pod %s-%s-%s-%s log websocket upgrade failed %s", clusterID, namespace, pod, container, err)
		return
	}
	defer conn.Close()

	readCloser, err := m.openPodLog(cluster, namespace, pod, container)
	if err != nil {
		log.Warnf("openPodLog %s-%s-%s-%s failed %s", clusterID, namespace, pod, container, err)
		return
	}

	s := bufio.NewScanner(readCloser)
	for s.Scan() {
		err := conn.WriteMessage(websocket.TextMessage, s.Bytes())
		if err != nil {
			log.Warnf("send log failed:%s", err.Error())
			break
		}
	}
}
