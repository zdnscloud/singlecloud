package handler

import (
	"encoding/json"
	"fmt"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	monitorNameSpace    = "zcloud-monitor1"
	monitorAppName      = "zcloud-monitor1"
	monitorChartName    = "prometheus"
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
	monitor := ctx.Object.(*types.Monitor)
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}
	url := genUrlPrefix(ctx, cluster.Name, monitorNameSpace)

	app, err := genMonitorApplication(monitor)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	app.SetID(app.Name)
	if err := m.apps.create(ctx, cluster.KubeClient, monitorNameSpace, url, app); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("create monitor application failed %s", err.Error()))
	}
	monitor.SetID(app.Name)
	return monitor, nil
}

func (m *MonitorManager) List(ctx *resttypes.Context) interface{} {
	monitors := []*types.Monitor{}
	monitor := &types.Monitor{
		IngressDomain:       "monitor.cluster.w",
		StorageClass:        "lvm",
		StorageSize:         50,
		PrometheusRetention: 10,
		ScrapeInterval:      15,
		AdminPassword:       "admin",
		RedirectUrl:         "http://monitor.cluster.w",
	}
	monitor.SetID(monitorAppName)
	monitors = append(monitors, monitor)
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
		Name:         monitorAppName,
		ChartName:    monitorChartName,
		ChartVersion: monitorChartVersion,
		Configs:      config,
	}, nil
}

func genMonitorConfigs(m *types.Monitor) (string, error) {
	p := charts.Prometheus{
		IngressDomain: []string{m.IngressDomain},
		AdminPassword: m.AdminPassword,
	}
	content, err := json.Marshal(&p)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
