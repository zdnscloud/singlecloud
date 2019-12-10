package clusteragent

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/zdnscloud/goproxy"
)

const (
	AgentKey                = "_agent_key"
	ClusterAgentServiceHost = "http://cluster-agent.zcloud.svc/apis/agent.zcloud.cn/v1"
	MethodGet               = "GET"
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

func (m *AgentManager) proxyRequest(cluster string, req *http.Request) (*http.Response, error) {
	dialer := m.server.GetAgentDialer(cluster, 15*time.Second)
	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer,
		},
	}

	resp, err := client.Do(req)
	return resp, err
}

func (m *AgentManager) processRequest(method, cluster, url string, resource interface{}) error {
	req, err := http.NewRequest(method, ClusterAgentServiceHost+url, nil)
	if err != nil {
		return err
	}
	resp, err := m.proxyRequest(cluster, req)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	return json.Unmarshal(body, resource)
}

func (m *AgentManager) GetResource(cluster, url string, resource interface{}) error {
	return m.processRequest(MethodGet, cluster, url, resource)
}

func (m *AgentManager) ListResource(cluster, url string, resources interface{}) error {
	var info Collection
	info.Data = resources
	if err := m.processRequest(MethodGet, cluster, url, &info); err != nil {
		return err
	}

	if info.Type != "collection" {
		return errors.New("url wrong, must resource collection")
	}
	return nil
}

type Collection struct {
	Type         string            `json:"type,omitempty"`
	ResourceType string            `json:"resourceType,omitempty"`
	Links        map[string]string `json:"links,omitempty"`
	Data         interface{}       `json:"data"`
}
