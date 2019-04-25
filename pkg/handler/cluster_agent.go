package handler

import (
	"net/http"
)

func (m *ClusterManager) RegisterAgent(clusterID, agentKey string, r *http.Request, w http.ResponseWriter) {
	cluster := m.get(clusterID)
	if cluster != nil {
		cluster.AgentManager.HandleAgentRegister(agentKey, r, w)
	}
}

func (m *ClusterManager) HandleAgentProxy(clusterID, agentKey, targetService string, r *http.Request, w http.ResponseWriter) {
	cluster := m.get(clusterID)
	if cluster != nil {
		cluster.AgentManager.HandleAgentProxy(agentKey, targetService, r, w)
	}
}
