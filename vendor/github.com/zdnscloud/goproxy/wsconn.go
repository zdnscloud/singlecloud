package goproxy

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	PingWaitDuration  = time.Duration(10 * time.Second)
	PingWriteInterval = time.Duration(5 * time.Second)
	MaxRead           = 8192
)

type wsConn struct {
	*websocket.Conn
	sync.Mutex

	pingCancel context.CancelFunc
	pingWait   sync.WaitGroup
}

func newWSConn(conn *websocket.Conn) *wsConn {
	w := &wsConn{
		Conn: conn,
	}
	w.setupDeadline()
	return w
}

func (w *wsConn) WriteMessage(typ int, data []byte) error {
	w.Lock()
	defer w.Unlock()
	w.Conn.SetWriteDeadline(time.Now().Add(PingWaitDuration))
	return w.Conn.WriteMessage(typ, data)
}

func (w *wsConn) WriteControl(typ int, data []byte, deadline time.Time) error {
	w.Lock()
	defer w.Unlock()
	return w.Conn.WriteControl(typ, data, deadline)
}

func (w *wsConn) setupDeadline() {
	w.SetReadDeadline(time.Now().Add(PingWaitDuration))
	w.SetPingHandler(func(string) error {
		w.WriteControl(websocket.PongMessage, []byte(""), time.Now().Add(time.Second))
		return w.SetReadDeadline(time.Now().Add(PingWaitDuration))
	})
	w.SetPongHandler(func(string) error {
		return w.SetReadDeadline(time.Now().Add(PingWaitDuration))
	})
}

func (w *wsConn) startPing() {
	ctx, cancel := context.WithCancel(context.Background())
	w.pingCancel = cancel
	w.pingWait.Add(1)

	go func() {
		defer w.pingWait.Done()
		t := time.NewTicker(PingWriteInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				w.WriteControl(websocket.PingMessage, []byte(""), time.Now().Add(time.Second))
			}
		}
	}()
}

func (w *wsConn) stopPing() {
	w.pingCancel()
	w.pingWait.Wait()
}
