package handler

import (
	"encoding/json"
	"fmt"
	"github.com/zdnscloud/singlecloud/pkg/zke"
	"math/rand"
	"strconv"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	monitorAppNamePrefix = "monitor"
	monitorChartName     = "prometheus"
	monitorChartVersion  = "6.4.1"
	monitorStorageClass  = "lvm"
	monitorStorageSize   = "10Gi"
	monitorAdminPassword = "zcloud"

	ZcloudDynamicaDomainPrefix  = "zc.zdns.cn"
	sysApplicationCheckInterval = time.Second * 5
	sysApplicationCheckTimes    = 30
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

	existApps, err := getApplicationsFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), monitorChartName)

	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("get monitor application from db failed %s", err.Error()))
	}
	if len(existApps) > 0 {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("cluster %s monitor has exist", cluster.Name))
	}

	requiredStorageClass := monitorStorageClass
	if len(monitor.StorageClass) > 0 {
		requiredStorageClass = monitor.StorageClass
	}

	if !isStorageClassExist(cluster.KubeClient, requiredStorageClass) {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("%s storageclass does't exist in cluster %s", requiredStorageClass, cluster.Name))
	}

	app, err := genMonitorApplication(cluster.KubeClient, monitor, cluster.Name)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	app.SetID(app.Name)

	if err := m.apps.create(ctx, cluster, ZCloudNamespace, app); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("create monitor application failed %s", err.Error()))
	}

	go ensureApplicationSucceedOrDie(ctx, m.clusters.GetDB(), cluster, storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), app.Name)

	monitor.Status = types.AppStatusCreate
	monitor.SetID(monitorAppNamePrefix)
	monitor.SetCreationTimestamp(time.Now())
	return monitor, nil
}

func (m *MonitorManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	monitors := []*types.Monitor{}

	apps, err := getApplicationsFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), monitorChartName)
	if err != nil || len(apps) == 0 {
		return nil
	}

	monitor, err := genMonitorFromApp(ctx, cluster.Name, apps[0])
	if err != nil {
		return nil
	}

	monitors = append(monitors, monitor)
	return monitors
}

func (m *MonitorManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	apps, err := getApplicationsFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), monitorChartName)
	if err != nil || len(apps) == 0 {
		return nil
	}

	monitor, err := genMonitorFromApp(ctx, cluster.Name, apps[0])
	if err != nil {
		return nil
	}

	return monitor
}

func genMonitorApplication(cli client.Client, m *types.Monitor, clusterName string) (*types.Application, error) {
	config, err := genMonitorConfigs(cli, m, clusterName)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         monitorAppNamePrefix + "-" + genRandomStr(12),
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
		m.IngressDomain = monitorAppNamePrefix + "-" + ZCloudNamespace + "-" + clusterName + "." + firstEdgeNodeIP + "." + ZcloudDynamicaDomainPrefix
	}
	m.RedirectUrl = "http://" + m.IngressDomain

	p := charts.Prometheus{
		Grafana: charts.PrometheusGrafana{
			Ingress: charts.PrometheusGrafanaIngress{
				Hosts: m.IngressDomain,
			},
			AdminPassword: monitorAdminPassword,
		},
		Prometheus: charts.PrometheusPrometheus{
			PrometheusSpec: charts.PrometheusSpec{
				StorageClass: monitorStorageClass,
				StorageSize:  monitorStorageSize,
			},
		},
		AlertManager: charts.PrometheusAlertManager{
			AlertManagerSpec: charts.AlertManagerSpec{
				StorageClass: monitorStorageClass,
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
	if len(m.AdminPassword) > 0 {
		p.Grafana.AdminPassword = m.AdminPassword
	}

	content, err := json.Marshal(&p)
	if err != nil {
		return nil, err
	}
	return content, nil
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

func genMonitorFromApp(ctx *resttypes.Context, cluster string, app *types.Application) (*types.Monitor, error) {
	p := charts.Prometheus{}
	if err := json.Unmarshal(app.Configs, &p); err != nil {
		return nil, err
	}
	m := types.Monitor{
		IngressDomain: p.Grafana.Ingress.Hosts,
		StorageClass:  p.Prometheus.PrometheusSpec.StorageClass,
		RedirectUrl:   "http://" + p.Grafana.Ingress.Hosts,
		Status:        app.Status,
	}
	m.SetID(app.Name)
	m.CreationTimestamp = app.CreationTimestamp
	return &m, nil
}

func genRandomStr(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func ensureApplicationSucceedOrDie(ctx *resttypes.Context, db storage.DB, cluster *zke.Cluster, tableName, appName string) {
	for i := 0; i < sysApplicationCheckTimes; i++ {
		sysApp, err := getApplicationFromDB(db, tableName, appName)
		if err != nil {
			if err == storage.ErrNotFoundResource {
				log.Infof("delete system application %s succeed", appName)
				return
			} else {
				log.Warnf("get system application %s failed %s", appName, err.Error())
			}
		}
		switch sysApp.Status {
		case types.AppStatusFailed:
			app, err := updateApplicationStatusFromDB(db, getCurrentUser(ctx), tableName, appName, types.AppStatusDelete)
			if err != nil {
				log.Warnf("delete system application %s failed %s", appName, err.Error())
				return
			}
			go deleteApplication(db, cluster.KubeClient, tableName, ZCloudNamespace, app)
		case types.AppStatusSucceed:
			log.Infof("create system application %s succeed", appName)
			return
		}
		time.Sleep(sysApplicationCheckInterval)
	}
}
