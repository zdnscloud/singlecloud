package clusteragent

import (
	"github.com/gin-gonic/gin"
)

const (
	ClusterAgentPrefix       = "/apis/agent.zcloud.cn/v1"
	ClusterAgentRegisterPath = ClusterAgentPrefix + "/register/:agentKey"
)

var (
	ClusterAgentProxyPaths = []string{
		ClusterAgentPrefix + "/clusters/:cluster/podnetworks",
		ClusterAgentPrefix + "/clusters/:cluster/nodenetworks",
		ClusterAgentPrefix + "/clusters/:cluster/servicenetworks",
		ClusterAgentPrefix + "/clusters/:cluster/namespaces/:namespace/innerservices",
		ClusterAgentPrefix + "/clusters/:cluster/namespaces/:namespace/outerservices",
	}
)

func (m *AgentManager) RegisterHandler(router gin.IRoutes) error {
	router.GET(ClusterAgentRegisterPath, func(c *gin.Context) {
		m.HandleAgentRegister(c.Param("agentKey"), c.Request, c.Writer)
	})

	for _, path := range ClusterAgentProxyPaths {
		router.GET(path, func(c *gin.Context) {
			m.HandleAgentProxy(c.Param("cluster"), c.Request, c.Writer)
		})
	}

	return nil
}
