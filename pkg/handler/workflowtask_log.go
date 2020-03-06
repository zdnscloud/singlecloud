package handler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"
)

type workFlowTaskContainer struct {
	pod       string
	container string
}

func (m *ClusterManager) OpenWorkFlowTaskLog(clusterID, namespace, workFlow, workFlowTask string, r *http.Request, w http.ResponseWriter) {
	cluster := m.GetClusterByName(clusterID)
	if cluster == nil {
		log.Infof("cluster %s isn't found to open workflowtask %s_%s_%s log", clusterID, namespace, workFlow, workFlowTask)
		return
	}

	_, err := getWorkFlowTask(cluster.GetKubeClient(), namespace, workFlowTask)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Infof("workflowtask %s_%s_%s_%s doesn't exist to open log", clusterID, namespace, workFlow, workFlowTask)
		} else {
			log.Warnf("get workflowtask %s_%s_%s_%s failed to open log", clusterID, namespace, workFlow, workFlowTask)
		}
		return
	}

	conn, err := websocket.Upgrade(w, r, nil, 4096, 4096)
	if err != nil {
		log.Warnf("workflowtask %s_%s_%s_%s log websocket upgrade failed %s", clusterID, namespace, workFlow, workFlowTask, err.Error())
		return
	}
	defer conn.Close()

	readedContainers := []workFlowTaskContainer{}
	for {
		allContainers, err := getNewWorkFlowContainers(cluster.GetKubeClient(), namespace, workFlowTask)
		if err != nil {
			log.Warnf("get workflowtask %s_%s_%s_%s containers failed to open log %s", clusterID, namespace, workFlow, workFlowTask, err.Error())
			return
		}

		unreadContainers := getUnreadWorkFlowContainers(readedContainers, allContainers)
		if len(unreadContainers) == 0 {
			return
		}

		for _, container := range unreadContainers {
			err := readWorkFlowContainerLogToWs(cluster.GetKubeClient(), conn, namespace, container.pod, container.container)
			if err != nil {
				if err == io.EOF {
					readedContainers = append(readedContainers, container)
					continue
				}
				log.Warnf("read workflowtask %s_%s_%s_%s container %s_%s log failed %s", clusterID, namespace, workFlow, workFlowTask, container.pod, container.container, err.Error())
				return
			}
		}
	}
	return
}

func openWorkFlowContainerLog(cli client.Client, namespace, pod, container string) (io.ReadCloser, error) {
	podClient, _ := cli.RestClientForObject(&corev1.Pod{}, 0)
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
	return req.Stream()
}

func readWorkFlowContainerLogToWs(cli client.Client, conn *websocket.Conn, namespace, pod, container string) error {
	readCloser, err := openWorkFlowContainerLog(cli, namespace, pod, container)
	if err != nil {
		return err
	}
	defer readCloser.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("=================%s====================", container))); err != nil {
		return err
	}

	s := bufio.NewScanner(readCloser)
	for {
		if !s.Scan() {
			return io.EOF
		}
		err := conn.WriteMessage(websocket.TextMessage, s.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func getNewWorkFlowContainers(cli client.Client, namespace, workFlowTaskName string) ([]workFlowTaskContainer, error) {
	wft, err := getWorkFlowTask(cli, namespace, workFlowTaskName)
	if err != nil {
		return nil, err
	}

	containers := []workFlowTaskContainer{}
	for _, task := range wft.SubTasks {
		if task.PodName != "" && len(task.Containers) > 0 {
			for _, c := range task.Containers {
				containers = append(containers, workFlowTaskContainer{
					pod:       task.PodName,
					container: c,
				})
			}
		}
	}
	return containers, nil
}

func getUnreadWorkFlowContainers(readed, all []workFlowTaskContainer) []workFlowTaskContainer {
	result := []workFlowTaskContainer{}
	for _, c1 := range all {
		var found bool
		for _, c2 := range readed {
			if c1.pod == c2.pod && c1.container == c2.container {
				found = true
				break
			}
		}
		if !found {
			result = append(result, c1)
		}
	}
	return result
}
