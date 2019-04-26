package clusteragent

import (
	"io"
	"net/http"
	"time"

	"github.com/zdnscloud/cement/log"
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

func (m *AgentManager) HandleAgentProxy(cluster string, r *http.Request, w http.ResponseWriter) {
	url := "http://cluster-agent.zcloud.svc.cluster.local"
	log.Infof("--> proxy path: %s", r.URL.Path)
	return

	proxyReq, err := http.NewRequest(r.Method, url, nil)
	proxyReq.Header = make(http.Header)
	for h, val := range r.Header {
		proxyReq.Header[h] = val
	}

	dialer := m.server.GetAgentDialer(cluster, 15*time.Second)
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
