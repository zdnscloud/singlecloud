package clusteragent

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/zdnscloud/goproxy"
)

const (
	AgentKey = "_agent_key"
)

type AgentManager struct {
	server *goproxy.Server
}

func New() *AgentManager {
	return &AgentManager{
		server: goproxy.New(authorizer),
	}
}

func authorizer(req *http.Request) (string, bool, error) {
	agentKey := req.Header.Get(AgentKey)
	return agentKey, agentKey != "", nil
}

func (m *AgentManager) HandleAgentRegister(agentKey string, r *http.Request, w http.ResponseWriter) {
	r.Header.Add(AgentKey, agentKey)
	m.server.ServeHTTP(w, r)
}

func (m *AgentManager) HandleAgentProxy(agentKey, targetService string, r *http.Request, w http.ResponseWriter) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}

	url := "http://" + targetService
	proxyReq, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
	proxyReq.Header = make(http.Header)
	for h, val := range r.Header {
		proxyReq.Header[h] = val
	}

	dialer := m.server.GetAgentDialer(agentKey, 15*time.Second)
	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer,
		},
	}
	resp, err := client.Do(proxyReq)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
