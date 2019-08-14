package handler

import (
	"encoding/json"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	monitorNameSpace    = "zcloud"
	monitorAppName      = "zcloud-registry"
	monitorChartName    = "prometheus-operator"
	monitorChartVersion = "6.4.1"
)

type MonitorManager struct {
	api.DefaultHandler
	clusters *ClusterManager
	apps     *ApplicationManager
}

func newMonitorManager(clusterMgr *ClusterManager, appMgr *ApplicationManager) *MonitorManager {
	return &MonitorManager{
		clusters: clusterMgr,
		apps:     appMgr,
	}
}

func (m *MonitorManager) Create(ctx *resttypes.Context, yaml []byte) (interface{}, *resttypes.APIError) {
	return nil, nil
}

func (m *MonitorManager) Get(ctx *resttypes.Context) interface{} {
	return nil
}

func (m *MonitorManager) List(ctx *resttypes.Context) interface{} {
	monitors := []*types.Monitor{}
	return monitors
}

func (m *MonitorManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	return nil
}

func genMonitorApplication(m *types.Monitor) (*types.Application, error) {
	config, err := genMonitorConfigs(m)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Configs: config,
	}, nil
}

func genMonitorConfigs(m *types.Monitor) (string, error) {
	harbor := charts.Harbor{}
	content, err := json.Marshal(&harbor)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
