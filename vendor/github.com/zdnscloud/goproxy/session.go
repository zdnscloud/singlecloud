package goproxy

import (
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Session struct {
	nextConnID int64
	conn       *wsConn
	mu         sync.Mutex
	conns      map[int64]*connection
	auth       ConnectAuthorizer
	isAgent    bool
}

func NewAgentSession(auth ConnectAuthorizer, conn *websocket.Conn) *Session {
	return &Session{
		nextConnID: 1,
		conn:       newWSConn(conn),
		conns:      map[int64]*connection{},
		auth:       auth,
		isAgent:    true,
	}
}

func newSession(agentKey string, conn *websocket.Conn) *Session {
	return &Session{
		nextConnID: 1,
		conn:       newWSConn(conn),
		conns:      map[int64]*connection{},
	}
}

func (s *Session) Serve() (int, error) {
	if s.isAgent {
		s.conn.startPing()
	}

	for {
		typ, reader, err := s.conn.NextReader()
		if err != nil {
			return 400, err
		}

		if typ != websocket.BinaryMessage {
			return 400, errWrongMessageType
		}

		if err := s.serveMessage(reader); err != nil {
			return 500, err
		}
	}
}

func (s *Session) serveMessage(reader io.Reader) error {
	message, err := newServerMessage(reader)
	if err != nil {
		return err
	}

	if message.messageType == Connect {
		if s.auth == nil || !s.auth(message.proto, message.address) {
			return errors.New("connect not allowed")
		}
		s.clientConnect(message)
		return nil
	}

	s.mu.Lock()
	conn := s.conns[message.connID]
	s.mu.Unlock()

	if conn == nil {
		if message.messageType == Data {
			newErrorMessage(message.connID, errUnknownConnection).WriteTo(s.conn)
		}
		return nil
	}

	switch message.messageType {
	case Data:
		if _, err := io.Copy(conn.GetMessageWriter(), message); err != nil {
			s.closeConnection(message.connID, err)
		}
	case Error:
		s.closeConnection(message.connID, message.err)
	}

	return nil
}

func (s *Session) closeConnection(connID int64, err error) {
	s.mu.Lock()
	conn := s.conns[connID]
	delete(s.conns, connID)
	s.mu.Unlock()

	if conn != nil {
		if err != nil {
			conn.reportErr(err)
		}
		conn.doClose()
	}
}

func (s *Session) clientConnect(message *message) {
	conn := newConnection(message.connID, s, message.proto, message.address)
	s.mu.Lock()
	s.conns[message.connID] = conn
	s.mu.Unlock()
	go proxyRealService(conn, message)
}

func (s *Session) getDialer(deadline time.Duration) Dialer {
	return func(proto, address string) (net.Conn, error) {
		return s.createConnectionForClient(deadline, proto, address)
	}
}

func (s *Session) createConnectionForClient(deadline time.Duration, proto, address string) (net.Conn, error) {
	connID := atomic.AddInt64(&s.nextConnID, 1)
	conn := newConnection(connID, s, proto, address)

	s.mu.Lock()
	s.conns[connID] = conn
	s.mu.Unlock()

	_, err := s.writeMessage(newConnect(connID, deadline, proto, address))
	if err != nil {
		s.closeConnection(connID, err)
		return nil, err
	}

	return conn, err
}

func (s *Session) writeMessage(message *message) (int, error) {
	return message.WriteTo(s.conn)
}

func (s *Session) Close() {
	if s.isAgent {
		s.conn.stopPing()
	}

	s.mu.Lock()
	for _, conn := range s.conns {
		conn.reportErr(io.EOF)
		conn.doClose()
	}
	s.conns = map[int64]*connection{}
	s.mu.Unlock()
}
