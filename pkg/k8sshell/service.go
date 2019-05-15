package k8sshell

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/exec"
	"github.com/zdnscloud/singlecloud/hack/sockjs"
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
	clusterShellPath := fmt.Sprintf(WSClusterShellPathTemp, ":cluster") + "/*actions"
	router.GET(clusterShellPath, func(c *gin.Context) {
		mgr.OpenClusterConsole(c.Param("cluster"), c.Request, c.Writer)
	})

	containerShellPath := fmt.Sprintf(WSClusterShellPathTemp, ":cluster", ":namespace", ":pod", ":container") + "/*actions"
	router.GET(containerShellPath, func(c *gin.Context) {
		mgr.OpenContainerConsole(c.Param("cluster"), c.Param("namespace"), c.Param("pod"), c.Param("container"), c.Request, c.Writer)
	})

	return nil
}

const (
	ClusterShellPodName  = "zcloud-shell"
	ClusterShellPodImage = "zdnscloud/kubectl:v1.13.4"
)

var _ io.ReadWriter = &ShellConn{}
var _ remotecommand.TerminalSizeQueue = &ShellConn{}

type ShellConn struct {
	conn     sockjs.Session
	sizeChan chan *remotecommand.TerminalSize
}

func newShellConn(session sockjs.Session) *ShellConn {
	return &ShellConn{
		conn:     session,
		sizeChan: make(chan *remotecommand.TerminalSize),
	}
}

func (t *ShellConn) Read(p []byte) (int, error) {
	var reply string
	var msg map[string]uint16
	reply, err := t.conn.Recv()
	if err != nil {
		return 0, err
	}
	if err := json.Unmarshal([]byte(reply), &msg); err != nil {
		return copy(p, reply), nil
	} else {
		t.sizeChan <- &remotecommand.TerminalSize{
			msg["cols"],
			msg["rows"],
		}
		return 0, nil
	}
}

func (t *ShellConn) Write(p []byte) (int, error) {
	return len(p), t.conn.Send(string(p))
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

	Sockjshandler := func(session sockjs.Session) {
		cmd := exec.Cmd{
			Path: BashPath,
		}

		pod := exec.Pod{
			Namespace:          handler.ZCloudNamespace,
			Name:               ClusterShellPodName,
			Container:          ClusterShellPodName,
			Image:              ClusterShellPodImage,
			ServiceAccountName: handler.ZCloudReadonly,
		}

		if err := executor.CreatePod(pod, cmd, 30*time.Second); err != nil {
			log.Errorf("execute cmd failed %s", err.Error())
			return
		}

		stream := newShellConn(session)
		err := executor.Exec(pod, cmd, stream)
		if err != nil {
			log.Errorf("execute cmd failed %s", err.Error())
		}
	}

	path := fmt.Sprintf(WSClusterShellPathTemp, clusterID)
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}

func (mgr *ExecutorManager) OpenContainerConsole(clusterID, namespace, pod, container string, r *http.Request, w http.ResponseWriter) {
	mgr.lock.Lock()
	executor, ok := mgr.executors[clusterID]
	mgr.lock.Unlock()

	if ok == false {
		log.Warnf("cluster %s is unknow for shell executor", clusterID)
		return
	}

	Sockjshandler := func(session sockjs.Session) {
		pod := exec.Pod{
			Namespace: namespace,
			Name:      pod,
			Container: container,
		}

		stream := newShellConn(session)
		if err := executor.Exec(pod, exec.Cmd{Path: BashPath}, stream); err != nil {
			log.Errorf("execute bash failed %s", err.Error())
		} else if err := executor.Exec(pod, exec.Cmd{Path: ShPath}, stream); err != nil {
			log.Errorf("execute sh failed %s", err.Error())
		}
	}

	path := fmt.Sprintf(WSContainerShellPathTemp, clusterID)
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
