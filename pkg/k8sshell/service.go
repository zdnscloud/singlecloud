package k8sshell

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/exec"
	"github.com/zdnscloud/singlecloud/pkg/handler"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	WSPrefix                 = "/apis/ws.zcloud.cn/v1"
	WSClusterShellPathTemp   = WSPrefix + "/clusters/%s/shell"
	WSContainerShellPathTemp = WSPrefix + "/clusters/%s/namespaces/%s/pods/%s/containers/%s/shell"

	BashPath = "/bin/bash"
	ShPath   = "/bin/sh"
)

func (mgr *ExecutorManager) RegisterHandler(router gin.IRoutes) error {
	clusterShellPath := fmt.Sprintf(WSClusterShellPathTemp, ":cluster")
	router.GET(clusterShellPath, func(c *gin.Context) {
		mgr.OpenClusterConsole(c.Param("cluster"), c.Request, c.Writer)
	})

	containerShellPath := fmt.Sprintf(WSContainerShellPathTemp, ":cluster", ":namespace", ":pod", ":container")
	router.GET(containerShellPath, func(c *gin.Context) {
		mgr.OpenContainerConsole(c.Param("cluster"), c.Param("namespace"), c.Param("pod"), c.Param("container"), c.Request, c.Writer)
	})

	return nil
}

const (
	ClusterShellPodName       = "zcloud-shell-0"
	ClusterShellContainerName = "zcloud-shell"
)

var _ io.ReadWriter = &ShellConn{}
var _ remotecommand.TerminalSizeQueue = &ShellConn{}

type ShellConn struct {
	conn     *websocket.Conn
	sizeChan chan *remotecommand.TerminalSize
}

func newShellConn(r *http.Request, w http.ResponseWriter) (*ShellConn, error) {
	var upgrader = websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &ShellConn{
		conn:     conn,
		sizeChan: make(chan *remotecommand.TerminalSize),
	}, nil
}

func (t *ShellConn) Read(p []byte) (int, error) {
	var msg map[string]uint16
	_, reply, err := t.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	if err := json.Unmarshal(reply, &msg); err != nil {
		return copy(p, string(reply)), nil
	} else {
		t.sizeChan <- &remotecommand.TerminalSize{
			msg["cols"],
			msg["rows"],
		}
		return 0, nil
	}
}

func (t *ShellConn) Write(p []byte) (int, error) {
	return len(p), t.conn.WriteMessage(websocket.TextMessage, p)
}

func (t *ShellConn) Next() *remotecommand.TerminalSize {
	return <-t.sizeChan
}

func (mgr *ExecutorManager) OpenClusterConsole(clusterID string, r *http.Request, w http.ResponseWriter) {
	mgr.lock.Lock()
	executor, ok := mgr.executors[clusterID]
	mgr.lock.Unlock()

	if ok == false {
		log.Warnf("cluster %s is unknow for shell executor", clusterID)
		return
	}

	cmd := exec.Cmd{
		Path: BashPath,
	}

	pod := exec.Pod{
		Namespace: handler.ZCloudNamespace,
		Name:      ClusterShellPodName,
		Container: ClusterShellContainerName,
	}

	stream, err := newShellConn(r, w)
	if err != nil {
		log.Warnf("new cluster %s console conn failed %s", clusterID, err.Error())
	}
	defer stream.conn.Close()

	if err := executor.Exec(pod, cmd, stream); err != nil {
		log.Errorf("execute cmd failed %s", err.Error())
	}
}

func (mgr *ExecutorManager) OpenContainerConsole(clusterID, namespace, pod, container string, r *http.Request, w http.ResponseWriter) {
	mgr.lock.Lock()
	executor, ok := mgr.executors[clusterID]
	mgr.lock.Unlock()

	if ok == false {
		log.Warnf("cluster %s is unknow for shell executor", clusterID)
		return
	}

	k8sPod := exec.Pod{
		Namespace: namespace,
		Name:      pod,
		Container: container,
	}

	stream, err := newShellConn(r, w)
	if err != nil {
		log.Warnf("new container %s-%s-%s-%s console failed %s", clusterID, namespace, pod, container, err.Error())
	}
	defer stream.conn.Close()

	if err := executor.Exec(k8sPod, exec.Cmd{Path: ShPath}, stream); err != nil {
		log.Errorf("execute bash failed %s", err.Error())
	}
}
