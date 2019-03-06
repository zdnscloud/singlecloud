package k8smanager

import (
	"encoding/json"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"k8s.io/client-go/tools/remotecommand"
)

type TerminalSockjs struct {
	conn     sockjs.Session
	sizeChan chan *remotecommand.TerminalSize
}

func (t *TerminalSockjs) Read(p []byte) (int, error) {
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

func (t *TerminalSockjs) Write(p []byte) (int, error) {
	return len(p), t.conn.Send(string(p))
}

func (t *TerminalSockjs) Next() *remotecommand.TerminalSize {
	return <-t.sizeChan
}
