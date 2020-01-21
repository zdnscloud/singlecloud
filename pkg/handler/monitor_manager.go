package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/randomdata"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/kvzoo"
)

const (
	monitorAppNamePrefix = "monitor"
	monitorChartName     = "prometheus"
	monitorChartVersion  = "6.4.1"

	ZcloudDynamicaDomainPrefix  = "zc.zdns.cn"
	sysApplicationCheckInterval = time.Second * 5
	sysApplicationCheckTimes    = 30
)

type MonitorManager struct {
	clusters *ClusterManager
	apps     *ApplicationManager
}

func newMonitorManager(clusterMgr *ClusterManager, appMgr *ApplicationManager) *MonitorManager {
	return &MonitorManager{
		clusters: clusterMgr,
		apps:     appMgr,
	}
}

func (m *MonitorManager) Create(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can enable cluster monitor")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	monitor := ctx.Resource.(*types.Monitor)
	app, err := genMonitorApplication(cluster, monitor)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, err.Error())
	}

	if err := createSysApplication(ctx, m.apps, cluster, monitorChartName, app, monitor.StorageClass, monitorAppNamePrefix); err != nil {
		return nil, err
	}

	monitor.Status = types.AppStatusCreate
	monitor.SetID(monitorAppNamePrefix)
	return monitor, nil
}

func createSysApplication(ctx *restresource.Context, appManager *ApplicationManager, cluster *zke.Cluster, chartName string, app *types.Application, requiredStorageClass string, appNamePrefix string) *resterr.APIError {
	table, _, err := createOrGetAppTable(cluster.Name, ZCloudNamespace)
	if err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get %s application from db failed %s", chartName, err.Error()))
	}

	existApp, err := getApplicationFromTableByChartName(table, chartName)
	if err != nil {
		log.Warnf("get cluster %s application by chart name %s failed %s", cluster.Name, chartName, err.Error())
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get %s application from db failed %s", chartName, err.Error()))
	}
	if existApp != nil {
		return resterr.NewAPIError(resterr.DuplicateResource, fmt.Sprintf("cluster %s %s has exist", cluster.Name, chartName))
	}

	if !isStorageClassExist(cluster.KubeClient, requiredStorageClass) {
		return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("%s storageclass does't exist in cluster %s", requiredStorageClass, cluster.Name))
	}

	app.Name = genAppNameIfDuplicate(table, app.Name, appNamePrefix)
	app.SetID(app.Name)

	if err := appManager.createApplication(ctx, cluster, ZCloudNamespace, app); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("create %s application failed %s", chartName, err.Error()))
	}

	go ensureApplicationSucceedOrDie(table, cluster, app.Name)
	return nil
}

func (m *MonitorManager) List(ctx *restresource.Context) interface{} {
	monitor := m.get(ctx)
	if monitor == nil {
		return nil
	} else {
		return []*types.Monitor{monitor.(*types.Monitor)}
	}
}

func (m *MonitorManager) Get(ctx *restresource.Context) restresource.Resource {
	id := ctx.Resource.GetID()
	if id != monitorAppNamePrefix {
		return nil
	}
	return m.get(ctx)
}

func (m *MonitorManager) get(ctx *restresource.Context) restresource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	app, err := getApplicationFromDBByChartName(cluster.Name, monitorChartName)
	if err != nil {
		log.Warnf("get cluster %s application by chart name %s failed %s", cluster.Name, monitorChartName, err.Error())
		return nil
	}
	if app == nil {
		return nil
	}

	monitor, err := genRetrunMonitorFromApplication(cluster.Name, app)
	if err != nil {
		return nil
	}
	return monitor
}

func (m *MonitorManager) Delete(ctx *restresource.Context) *resterr.APIError {
	if ctx.Resource.GetID() != monitorAppNamePrefix {
		return resterr.NewAPIError(resterr.NotFound, "monitor doesn't exist")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	return deleteApplicationByChartName(cluster, monitorChartName)
}

func deleteApplicationByChartName(cluster *zke.Cluster, chartName string) *resterr.APIError {
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}
	table, _, err := createOrGetAppTable(cluster.Name, ZCloudNamespace)
	if err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get cluster %s application %s from db failed %s", cluster.Name, chartName, err.Error()))
	}

	app, err := getApplicationFromTableByChartName(table, chartName)
	if err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get cluster %s application %s from db failed %s", cluster.Name, chartName, err.Error()))
	}
	if app == nil {
		return resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("%s doesn't exist", chartName))
	}

	appName := app.Name
	if err := deleteApplication(table, cluster, ZCloudNamespace, appName, true); err != nil {
		if err == kvzoo.ErrNotFound {
			return resterr.NewAPIError(resterr.NotFound,
				fmt.Sprintf("application %s with namespace %s doesn't exist", appName, ZCloudNamespace))
		} else {
			return resterr.NewAPIError(resterr.ServerError,
				fmt.Sprintf("delete application %s failed: %s", appName, err.Error()))
		}
	}
	return nil
}

func genMonitorApplication(cluster *zke.Cluster, m *types.Monitor) (*types.Application, error) {
	config, err := genMonitorApplicationConfig(cluster, m)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         monitorAppNamePrefix + "-" + randomdata.RandString(12),
		ChartName:    monitorChartName,
		ChartVersion: monitorChartVersion,
		Configs:      config,
		SystemChart:  true,
	}, nil
}

func genMonitorApplicationConfig(cluster *zke.Cluster, m *types.Monitor) ([]byte, error) {
	if len(m.IngressDomain) == 0 {
		edgeIP := getRandomEdgeNodeAddress(cluster)
		if len(edgeIP) == 0 {
			return nil, fmt.Errorf("can not find edge node for this cluster")
		}
		m.IngressDomain = monitorAppNamePrefix + "-" + ZCloudNamespace + "-" + cluster.Name + "." + edgeIP + "." + ZcloudDynamicaDomainPrefix
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
				StorageClass:   m.StorageClass,
				StorageSize:    m.StorageSize,
				Retention:      m.PrometheusRetention,
				ScrapeInterval: m.ScrapeInterval,
			},
		},
		AlertManager: charts.PrometheusAlertManager{
			AlertManagerSpec: charts.AlertManagerSpec{
				StorageClass: m.StorageClass,
			},
		},
		KubeEtcd: charts.PrometheusEtcd{
			Enabled:   true,
			EndPoints: cluster.GetNodeIpsByRole(types.RoleEtcd),
		},
	}

	return json.Marshal(&p)
}

func genRetrunMonitorFromApplication(cluster string, app *types.Application) (*types.Monitor, error) {
	p := charts.Prometheus{}
	if err := json.Unmarshal(app.Configs, &p); err != nil {
		return nil, err
	}
	m := types.Monitor{
		IngressDomain:       p.Grafana.Ingress.Hosts,
		StorageClass:        p.Prometheus.PrometheusSpec.StorageClass,
		StorageSize:         p.Prometheus.PrometheusSpec.StorageSize,
		PrometheusRetention: p.Prometheus.PrometheusSpec.Retention,
		ScrapeInterval:      p.Prometheus.PrometheusSpec.ScrapeInterval,
		RedirectUrl:         "http://" + p.Grafana.Ingress.Hosts,
		Status:              app.Status,
	}
	m.SetID(monitorAppNamePrefix)
	m.SetCreationTimestamp(time.Time(app.CreationTimestamp))
	m.SetDeletionTimestamp(time.Time(app.DeletionTimestamp))
	return &m, nil
}

func ensureApplicationSucceedOrDie(table kvzoo.Table, cluster *zke.Cluster, appName string) {
	for i := 0; i < sysApplicationCheckTimes; i++ {
		sysApp, err := getApplicationFromDB(table, appName, true)
		if err != nil {
			log.Warnf("get system application %s failed %s", appName, err.Error())
			return
		}

		switch sysApp.Status {
		case types.AppStatusFailed:
			if err := deleteApplication(table, cluster, ZCloudNamespace, appName, true); err != nil {
				log.Warnf("delete system application %s failed %s", appName, err.Error())
				return
			}
		case types.AppStatusSucceed:
			log.Infof("create system application %s succeed", appName)
			return
		}
		time.Sleep(sysApplicationCheckInterval)
	}
}

func getApplicationFromDBByChartName(clusterName, chartName string) (*types.Application, error) {
	table, _, err := createOrGetAppTable(clusterName, ZCloudNamespace)
	if err != nil {
		return nil, err
	}

	return getApplicationFromTableByChartName(table, chartName)
}

func getApplicationFromTableByChartName(table kvzoo.Table, chartName string) (*types.Application, error) {
	apps, err := getApplicationsFromDB(table)
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		if app.ChartName == chartName {
			return app, nil
		}
	}
	return nil, nil
}

func genAppNameIfDuplicate(table kvzoo.Table, appName, appNamePrefex string) string {
	for {
		duplicateApp, _ := getApplicationFromDB(table, appName, true)
		if duplicateApp != nil {
			appName = appNamePrefex + "-" + randomdata.RandString(12)
		} else {
			break
		}
	}

	return appName
}
