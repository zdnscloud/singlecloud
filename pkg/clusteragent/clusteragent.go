package clusteragent

import (
	"encoding/json"
	"github.com/zdnscloud/goproxy"
	resttypes "github.com/zdnscloud/gorest/resource"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	AgentKey                = "_agent_key"
	ClusterAgentServiceHost = "http://cluster-agent.zcloud.svc"
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

func (m *AgentManager) GetData(cluster, url string) (interface{}, error) {
	req, err := http.NewRequest("GET", ClusterAgentServiceHost+url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.ProxyRequest(cluster, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var info resttypes.Collection
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &info)
	return info.Data, nil
}
