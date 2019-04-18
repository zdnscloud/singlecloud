package clusteragent

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/goproxy"
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
	agentKey := req.Header.Get("_agent_id")
	return agentKey, agentKey != "", nil
}

func (m *AgentManager) HandleAgentRegister(c *gin.Context) {
	m.server.ServeHTTP(c.Writer, c.Request)
}

func (m *AgentManager) HandleAgentProxy(c *gin.Context) {
	rw := c.Writer
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		rw.Write([]byte(err.Error()))
		rw.WriteHeader(500)
		return
	}

	url := "http:/" + c.Param("realservice")
	proxyReq, err := http.NewRequest(c.Request.Method, url, bytes.NewReader(body))
	proxyReq.Header = make(http.Header)
	for h, val := range c.Request.Header {
		proxyReq.Header[h] = val
	}

	agentKey := c.Param("id")
	dialer := m.server.GetAgentDialer(agentKey, 15*time.Second)
	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer,
		},
	}
	resp, err := client.Do(proxyReq)
	if err != nil {
		rw.Write([]byte(err.Error()))
		rw.WriteHeader(500)
		return
	}
	defer resp.Body.Close()
	rw.WriteHeader(resp.StatusCode)
	io.Copy(rw, resp.Body)
}
