package handler

import (
	"net/http"

	"github.com/zdnscloud/singlecloud/pkg/logger"
)

func (m *ClusterManager) GetOneNamespaceEvents(id, namespace string, r *http.Request, w http.ResponseWriter) {
	cluster := m.get(id)
	if cluster == nil {
		logger.Warn("cluster %s isn't found to open console", id)
		return
	}

	//events := cluster.EventWatcher.GetOneNamespaceEvents(namespace)
}

func (m *ClusterManager) GetAllNamespaceEvents(id string, r *http.Request, w http.ResponseWriter) {
	cluster := m.get(id)
	if cluster == nil {
		logger.Warn("cluster %s isn't found to open console", id)
		return
	}

	// events := cluster.EventWatcher.GetAllNamespaceEvents()
}
