package clusteragent

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zdnscloud/goproxy"
)

const (
	AgentKey                = "_agent_key"
	ClusterAgentServiceHost = "http://cluster-agent.zcloud.svc"
	ContentTypeKey          = "Content-Type"
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
	newPath := strings.Replace(r.URL.Path, "/clusters/"+cluster, "", 1)
	proxyReq, err := http.NewRequest(r.Method, ClusterAgentServiceHost+newPath, nil)
	proxyReq.Header = make(http.Header)
	for h, val := range r.Header {
		proxyReq.Header[h] = val
	}

	resp, err := m.ProxyRequest(cluster, proxyReq)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}
	defer resp.Body.Close()
	w.Header().Set(ContentTypeKey, resp.Header.Get(ContentTypeKey))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (m *AgentManager) ProxyRequest(cluster string, req *http.Request) (*http.Response, error) {
	dialer := m.server.GetAgentDialer(cluster, 15*time.Second)
	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer,
		},
	}

	resp, err := client.Do(req)
	return resp, err
}
