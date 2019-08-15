package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	monitorNameSpace    = "zcloud"
	monitorAppName      = "zcloud-monitor"
	monitorChartName    = "prometheus"
	monitorChartVersion = "6.4.1"
	monitorTableName    = "cluster_monitor"
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

	app, err := genMonitorApplication(monitor)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	app.SetID(app.Name)
	if err := m.apps.create(ctx, cluster, monitorNameSpace, app); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("create monitor application failed %s", err.Error()))
	}
	monitor.SetID(app.Name)
	monitor.SetCreationTimestamp(time.Now())
	if err := m.addToDB(monitor); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("add monitor to db failed %s", err.Error()))
	}
	return monitor, nil
}

func (m *MonitorManager) List(ctx *resttypes.Context) interface{} {
	monitors := []*types.Monitor{}
	monitor, err := m.getFromDB()
	if err != nil {
		return monitors
	}
	monitors = append(monitors, monitor)
	return monitors
}

func (m *MonitorManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	app, err := updateApplicationStatusFromDB(m.clusters.GetDB(), monitorNameSpace, monitorAppName, types.AppStatusDelete)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("cluster monitor application %s doesn't exist", monitorAppName))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("delete cluster monitor application %s failed: %s", monitorAppName, err.Error()))
		}
	}
	if err := m.deleteFromDB(); err != nil {
		return resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("delete cluster monitor from db failed: %s", err.Error()))
	}
	go deleteApplication(m.clusters.GetDB(), cluster.KubeClient, genAppTableName(cluster.Name, monitorNameSpace), monitorNameSpace, app)
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

func genMonitorConfigs(m *types.Monitor) ([]byte, error) {
	p := charts.Prometheus{
		IngressDomain: []string{m.IngressDomain},
		AdminPassword: m.AdminPassword,
	}
	content, err := json.Marshal(&p)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (m *MonitorManager) addToDB(monitor *types.Monitor) error {
	value, err := json.Marshal(monitor)
	if err != nil {
		return fmt.Errorf("marshal monitor %s failed: %s", monitorAppName, err.Error())
	}

	tx, err := BeginTableTransaction(m.clusters.GetDB(), monitorTableName)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Add(monitorAppName, value); err != nil {
		return err
	}
	return tx.Commit()
}

func (m *MonitorManager) getFromDB() (*types.Monitor, error) {
	monitor := &types.Monitor{}
	tx, err := BeginTableTransaction(m.clusters.GetDB(), monitorTableName)
	if err != nil {
		return monitor, err
	}
	defer tx.Commit()

	value, err := tx.Get(monitorAppName)
	if err != nil {
		return monitor, err
	}

	err = json.Unmarshal(value, monitor)
	return monitor, err
}

func (m *MonitorManager) deleteFromDB() error {
	tx, err := BeginTableTransaction(m.clusters.GetDB(), monitorTableName)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Delete(monitorAppName); err != nil {
		return err
	}
	return tx.Commit()
}
