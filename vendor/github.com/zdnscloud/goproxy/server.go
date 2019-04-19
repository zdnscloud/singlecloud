package goproxy

import (
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"
)

type Dialer func(network, address string) (net.Conn, error)
type Authorizer func(req *http.Request) (agentKey string, authed bool, err error)

type Server struct {
	authorizer Authorizer
	sessions   *sessionManager
}

func New(auth Authorizer) *Server {
	return &Server{
		authorizer: auth,
		sessions:   newSessionManager(),
	}
}

func (s *Server) GetAgentDialer(agentKey string, deadline time.Duration) Dialer {
	return func(proto, address string) (net.Conn, error) {
		d, err := s.sessions.getAgentDialer(agentKey, deadline)
		if err != nil {
			return nil, err
		}
		return d(proto, address)
	}
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	s.registerAgent(rw, req)
}

func (s *Server) registerAgent(rw http.ResponseWriter, req *http.Request) {
	agentKey, authed, err := s.authorizer(req)
	if err != nil {
		log.Warnf("agent %s is authenticate failed %s", agentKey, err.Error())
		s.returnError(rw, req, 400, err)
		return
	}

	if !authed {
		log.Warnf("agent %s is reject", agentKey)
		s.returnError(rw, req, 401, errFailedAuth)
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
		CheckOrigin:      func(r *http.Request) bool { return true },
		Error:            s.returnError,
	}
	wsConn, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Errorf("upgrade conn to ws failed:%s", err.Error())
		s.returnError(rw, req, 400, err)
		return
	}

	session, err := s.sessions.addAgent(agentKey, wsConn)
	if err != nil {
		log.Errorf("add agent failed:%s", err.Error())
		s.returnError(rw, req, 400, err)
	}

	log.Infof("register agent with key %s", agentKey)
	session.Serve()
	s.sessions.removeAgent(agentKey)
	log.Infof("remove agent with key %s", agentKey)
}

func (s *Server) returnError(rw http.ResponseWriter, req *http.Request, code int, err error) {
	rw.Write([]byte(err.Error()))
	rw.WriteHeader(code)
}
