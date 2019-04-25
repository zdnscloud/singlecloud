package goproxy

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

const (
	WriteBufSize = 4096
)

type connection struct {
	wg sync.WaitGroup

	buf     chan []byte
	readBuf []byte
	addr    addr
	session *Session
	connID  int64
}

func newConnection(connID int64, session *Session, proto, address string) *connection {
	return &connection{
		addr: addr{
			proto:   proto,
			address: address,
		},
		connID:  connID,
		session: session,
		buf:     make(chan []byte, 1024),
	}
}

//this could be invoked by third party code
func (c *connection) Close() error {
	c.session.closeConnection(c.connID, io.EOF)
	return nil
}

//connection is managed by session, this is invoked by session
func (c *connection) doClose() {
	close(c.buf)
	c.wg.Wait()
}

func (c *connection) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}

	if len(c.readBuf) != 0 {
		n := copy(b, c.readBuf)
		c.readBuf = c.readBuf[n:]
		return n, nil
	}

	next, ok := <-c.buf
	if !ok {
		return 0, io.EOF
	}
	n := copy(b, next)
	if n < len(next) {
		c.readBuf = next[n:]
	}
	return n, nil
}

//get data from session
func (c *connection) GetMessageWriter() msgWriter {
	return msgWriter{
		conn: c,
		ch:   c.buf,
	}
}

func (c *connection) Write(b []byte) (int, error) {
	return c.session.writeMessage(newMessage(c.connID, 0, b))
}

func (c *connection) reportErr(err error) {
	if err != nil {
		c.session.writeMessage(newErrorMessage(c.connID, err))
	}
}

func (c *connection) LocalAddr() net.Addr {
	return c.addr
}

func (c *connection) RemoteAddr() net.Addr {
	return c.addr
}

func (c *connection) SetDeadline(t time.Time) error {
	return nil
}

func (c *connection) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *connection) SetWriteDeadline(t time.Time) error {
	return nil
}

type addr struct {
	proto   string
	address string
}

func (a addr) Network() string {
	return a.proto
}

func (a addr) String() string {
	return a.address
}

type msgWriter struct {
	conn *connection
	ch   chan []byte
}

func (w msgWriter) Write(buf []byte) (int, error) {
	cp := make([]byte, len(buf))
	copy(cp, buf)
	select {
	case w.ch <- cp:
		return len(buf), nil
	case <-time.After(15 * time.Second):
		return 0, errors.New("backed up reader")
	}
}
