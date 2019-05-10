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
	WSPrefix        = "/apis/ws.zcloud.cn/v1"
	WSShellPathTemp = WSPrefix + "/clusters/%s/shell"
)

func (mgr *ExecutorManager) RegisterHandler(router gin.IRoutes) error {
	shellPath := fmt.Sprintf(WSShellPathTemp, ":cluster") + "/*actions"
	router.GET(shellPath, func(c *gin.Context) {
		mgr.OpenConsole(c.Param("cluster"), c.Request, c.Writer)
	})

	return nil
}

const (
	ShellPodName  = "zcloud-shell"
	ShellPodImage = "zdnscloud/kubectl:v1.13.4"
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

func (mgr *ExecutorManager) OpenConsole(clusterID string, r *http.Request, w http.ResponseWriter) {
	mgr.lock.Lock()
	executor, ok := mgr.executors[clusterID]
	mgr.lock.Unlock()

	if ok == false {
		log.Warnf("cluster %s is unknow for shell executor", clusterID)
		return
	}

	Sockjshandler := func(session sockjs.Session) {
		cmd := exec.Cmd{
			Path: "/bin/bash",
		}

		pod := exec.Pod{
			Namespace:          handler.ZCloudNamespace,
			Name:               ShellPodName,
			Image:              ShellPodImage,
			ServiceAccountName: handler.ZCloudReadonly,
		}

		stream := newShellConn(session)
		err := executor.RunCmd(pod, cmd, stream, 30*time.Second)
		if err != nil {
			log.Errorf("execute cmd failed %s", err.Error())
		}
	}

	path := fmt.Sprintf(WSShellPathTemp, clusterID)
	sockjs.NewHandler(path, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
