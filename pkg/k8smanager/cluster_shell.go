package k8smanager

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/zdnscloud/gok8s/exec"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"k8s.io/client-go/tools/remotecommand"
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

func (m *ClusterManager) OpenConsole(id string, r *http.Request, w http.ResponseWriter) {
	cluster := m.Get(id)
	if cluster == nil {
		logger.Warn("cluster %s isn't found to open console", id)
		return
	}

	Sockjshandler := func(session sockjs.Session) {
		cmd := exec.Cmd{
			Path: "/bin/bash",
		}

		pod := exec.Pod{
			Namespace:          ZCloudNamespace,
			Name:               ShellPodName,
			Image:              ShellPodImage,
			ServiceAccountName: ZCloudReadonly,
		}

		stream := newShellConn(session)
		err := cluster.Executor.RunCmd(pod, cmd, stream, 30*time.Second)
		if err != nil {
			logger.Error("execute cmd failed %s", err.Error())
		}
	}

	sockjs.NewHandler("/zcloud/ws/clusters/"+id, sockjs.DefaultOptions, Sockjshandler).ServeHTTP(w, r)
}
