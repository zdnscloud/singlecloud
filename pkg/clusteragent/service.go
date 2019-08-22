package clusteragent

import (
	"github.com/gin-gonic/gin"
)

const (
	ClusterAgentPrefix       = "/apis/agent.zcloud.cn/v1"
	ClusterAgentRegisterPath = ClusterAgentPrefix + "/register/:agentKey"
)

func (m *AgentManager) RegisterHandler(router gin.IRoutes) error {
	router.GET(ClusterAgentRegisterPath, func(c *gin.Context) {
		m.HandleAgentRegister(c.Param("agentKey"), c.Request, c.Writer)
	})
	return nil
}
