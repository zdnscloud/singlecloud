package handler

import (
	"io"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/singlecloud/pkg/logger"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
)

func (m *ClusterManager) OpenPodLog(clusterID, namespace, pod, container string, r *http.Request, w http.ResponseWriter) {
	cluster := m.get(clusterID)
	if cluster == nil {
		logger.Warn("cluster %s isn't found to open console", clusterID)
		return
	}

	Sockjshandler := func(session sockjs.Session) {
		podClient, _ := cluster.KubeClient.RestClientForObject(&corev1.Pod{})
		opts := corev1.PodLogOptions{
			Follow:    true,
			Container: container,
		}
		req := podClient.
			Get().
			Namespace(namespace).
			Name(pod).
			Resource("pods").
			SubResource("log").
			VersionedParams(&opts, scheme.ParameterCodec)
		readCloser, err := req.Stream()
		wrapper := newShellConn(session)
		if err != nil {
			wrapper.Write([]byte(err.Error()))
			return
		}

		defer readCloser.Close()
		io.Copy(wrapper, readCloser)
	}

	path := strings.Join([]string{ShellClusterPrefix, clusterID, "namespaces", namespace, "pods", pod, "containers", container}, "/")
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
