package goproxy

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type sessionManager struct {
	sync.Mutex
	agents map[string]*Session
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		agents: make(map[string]*Session),
	}
}

func (m *sessionManager) getAgentDialer(agentKey string, deadline time.Duration) (Dialer, error) {
	m.Lock()
	defer m.Unlock()

	s, ok := m.agents[agentKey]
	if ok == false {
		return nil, fmt.Errorf("failed to find Session for client %s", agentKey)
	}

	return s.getDialer(deadline), nil
}

func (m *sessionManager) addAgent(agentKey string, conn *websocket.Conn) (*Session, error) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.agents[agentKey]; ok {
		return nil, fmt.Errorf("duplicate agent key %s from %v", agentKey, conn.RemoteAddr())
	}

	s := newSession(agentKey, conn)
	m.agents[agentKey] = s
	return s, nil
}

func (m *sessionManager) removeAgent(agentKey string) {
	m.Lock()
	defer m.Unlock()

	if s, ok := m.agents[agentKey]; ok {
		delete(m.agents, agentKey)
		s.Close()
	}
}
