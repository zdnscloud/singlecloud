package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	monitorNameSpace         = "zcloud"
	monitorAppName           = "zcloud-monitor"
	monitorChartName         = "prometheus"
	monitorChartVersion      = "6.4.1"
	monitorTableName         = "cluster_monitor"
	zcloudDynamicalDnsPrefix = "zc.zdns.cn"
	monitorAppStorageClass   = "lvm"
	monitorAppStorageSize    = "10Gi"
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
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can enable cluster monitor")
	}
	monitor := ctx.Object.(*types.Monitor)
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}
	if existMonitor, _ := m.getFromDB(cluster.Name); existMonitor != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "cluster monitor has exist")
	}

	if !isStorageClassExist(cluster.KubeClient, monitorAppStorageClass) {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("%s storageclass does't exist in cluster %s", monitorAppStorageClass, cluster.Name))
	}

	app, err := genMonitorApplication(cluster.KubeClient, monitor, cluster.Name)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	app.SetID(app.Name)
	if err := m.apps.create(ctx, cluster, monitorNameSpace, app); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("create monitor application failed %s", err.Error()))
	}
	monitor.SetID(app.Name)
	monitor.SetCreationTimestamp(time.Now())
	if err := m.addToDB(cluster.Name, monitor); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("add monitor to db failed %s", err.Error()))
	}
	return monitor, nil
}

func (m *MonitorManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}
	monitors := []*types.Monitor{}
	monitor, err := m.getFromDB(cluster.Name)
	if err != nil {
		return monitors
	}
	monitors = append(monitors, monitor)
	return monitors
}

func (m *MonitorManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can disable cluster monitor")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	if existMonitor, _ := m.getFromDB(cluster.Name); existMonitor == nil {
		return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("cluster %s monitor has exist", cluster.Name))
	}

	appTableName := storage.GenTableName(ApplicationTable, cluster.Name, monitorNameSpace)
	app, err := updateApplicationStatusFromDB(m.clusters.GetDB(), getCurrentUser(ctx), appTableName, monitorAppName, types.AppStatusDelete)
	if err != nil {
		if err == storage.ErrNotFoundResource {
			if err := m.deleteFromDB(cluster.Name); err != nil {
				return resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("delete cluster monitor from db failed: %s", err.Error()))
			}
			return nil
		} else {
			return resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("delete cluster %s monitor application %s failed: %s", cluster.Name, monitorAppName, err.Error()))
		}
	}
	go deleteApplication(m.clusters.GetDB(), cluster.KubeClient, appTableName, monitorNameSpace, app)

	if err := m.deleteFromDB(cluster.Name); err != nil {
		return resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("delete cluster monitor from db failed: %s", err.Error()))
	}
	return nil
}

func genMonitorApplication(cli client.Client, m *types.Monitor, clusterName string) (*types.Application, error) {
	config, err := genMonitorConfigs(cli, m, clusterName)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         monitorAppName,
		ChartName:    monitorChartName,
		ChartVersion: monitorChartVersion,
		Configs:      config,
		SystemChart:  true,
	}, nil
}

func genMonitorConfigs(cli client.Client, m *types.Monitor, clusterName string) ([]byte, error) {
	if len(m.IngressDomain) == 0 {
		firstEdgeNodeIP := getFirstEdgeNodeAddress(cli)
		if len(firstEdgeNodeIP) == 0 {
			return nil, fmt.Errorf("can not find edge node for this cluster")
		}
		m.IngressDomain = monitorAppName + "-" + monitorNameSpace + "-svc-" + clusterName + "." + firstEdgeNodeIP + "." + zcloudDynamicalDnsPrefix
	}
	m.RedirectUrl = "http://" + m.IngressDomain

	p := charts.Prometheus{
		Grafana: charts.PrometheusGrafana{
			Ingress: charts.PrometheusGrafanaIngress{
				Hosts: m.IngressDomain,
			},
			AdminPassword: m.AdminPassword,
		},
		Prometheus: charts.PrometheusPrometheus{
			PrometheusSpec: charts.PrometheusSpec{
				StorageClass: monitorAppStorageClass,
				StorageSize:  monitorAppStorageSize,
			},
		},
		AlertManager: charts.PrometheusAlertManager{
			AlertManagerSpec: charts.AlertManagerSpec{
				StorageClass: monitorAppStorageClass,
			},
		},
		KubeEtcd: charts.PrometheusEtcd{
			Enabled: true,
		},
	}

	etcds := getClusterEtcds(cli)
	if len(etcds) == 0 {
		return nil, fmt.Errorf("can not get etcd nodes info for this cluster")
	}
	p.KubeEtcd.EndPoints = etcds

	if m.PrometheusRetention > 0 {
		p.Prometheus.PrometheusSpec.Retention = strconv.Itoa(m.PrometheusRetention) + "d"
	}
	if m.ScrapeInterval > 0 {
		p.Prometheus.PrometheusSpec.ScrapeInterval = strconv.Itoa(m.ScrapeInterval) + "s"
	}
	if len(m.StorageClass) > 0 {
		p.AlertManager.AlertManagerSpec.StorageClass = m.StorageClass
		p.Prometheus.PrometheusSpec.StorageClass = m.StorageClass
	}
	if m.StorageSize > 0 {
		p.Prometheus.PrometheusSpec.StorageSize = strconv.Itoa(m.StorageSize) + "Gi"
	}

	content, err := json.Marshal(&p)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (m *MonitorManager) addToDB(clusterName string, monitor *types.Monitor) error {
	value, err := json.Marshal(monitor)
	if err != nil {
		return fmt.Errorf("marshal monitor %s failed: %s", monitorAppName, err.Error())
	}

	tx, err := BeginTableTransaction(m.clusters.GetDB(), monitorTableName)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Add(clusterName, value); err != nil {
		return err
	}
	return tx.Commit()
}

func (m *MonitorManager) getFromDB(clusterName string) (*types.Monitor, error) {
	monitor := &types.Monitor{}
	tx, err := BeginTableTransaction(m.clusters.GetDB(), monitorTableName)
	if err != nil {
		return nil, err
	}
	defer tx.Commit()

	value, err := tx.Get(clusterName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(value, monitor)
	return monitor, err
}

func (m *MonitorManager) deleteFromDB(clusterName string) error {
	tx, err := BeginTableTransaction(m.clusters.GetDB(), monitorTableName)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Delete(clusterName); err != nil {
		return err
	}
	return tx.Commit()
}

func getFirstEdgeNodeAddress(cli client.Client) string {
	nodes, err := getNodes(cli)
	if err != nil {
		return ""
	}
	for _, n := range nodes {
		if n.HasRole(types.RoleEdge) {
			return n.Address
		}
	}
	return ""
}

func getClusterEtcds(cli client.Client) []string {
	nodes, err := getNodes(cli)
	if err != nil {
		return nil
	}
	etcds := []string{}
	for _, n := range nodes {
		if n.HasRole(types.RoleEtcd) {
			etcds = append(etcds, n.Address)
		}
	}
	return etcds
}
