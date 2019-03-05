package k8smanager

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type connAaptor struct {
	conn   *websocket.Conn
	reader io.Reader
}

func newConnAdaptor(conn *websocket.Conn) *connAaptor {
	return &connAaptor{
		conn: conn,
	}
}

func (c *connAaptor) Close() error {
	return c.conn.Close()
}

func (c *connAaptor) Read(b []byte) (int, error) {
	if c.reader == nil {
		typ, reader, err := c.conn.NextReader()
		if err != nil {
			return 0, err
		}

		if typ != websocket.TextMessage {
			return 0, errors.New("only text message supported")
		}

		c.reader = reader
	}

	return c.reader.Read(b)
}

func (c *connAaptor) Write(b []byte) (int, error) {
	return len(b), c.conn.WriteMessage(websocket.TextMessage, b)
}

func (c *connAaptor) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *connAaptor) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *connAaptor) SetDeadline(t time.Time) error {
	return nil
}

func (c *connAaptor) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *connAaptor) SetWriteDeadline(t time.Time) error {
	return nil
}
