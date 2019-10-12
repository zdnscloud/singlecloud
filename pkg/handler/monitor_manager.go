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
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
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
	app, err := genMonitorApplication(cluster, monitor, cluster.Name)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, err.Error())
	}

	app.Name = genAppNameIfDuplicate(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), app.Name, monitorAppNamePrefix)
	app.SetID(app.Name)

	if err := createSysApplication(ctx, m.clusters.GetDB(), m.apps, cluster, monitorChartName, app, monitor.StorageClass); err != nil {
		return nil, err
	}

	monitor.Status = types.AppStatusCreate
	monitor.SetID(monitorAppNamePrefix)
	return monitor, nil
}

func createSysApplication(ctx *restresource.Context, db storage.DB, appManager *ApplicationManager, cluster *zke.Cluster, chartName string, app *types.Application, requiredStorageClass string) *resterr.APIError {
	tableName := storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace)

	hasExist, err := checkSysApplicationExist(db, tableName, chartName)
	if err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get %s application from db failed %s", chartName, err.Error()))
	}
	if hasExist {
		return resterr.NewAPIError(resterr.DuplicateResource, fmt.Sprintf("cluster %s %s has exist", cluster.Name, chartName))
	}

	if !isStorageClassExist(cluster.KubeClient, requiredStorageClass) {
		return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("%s storageclass does't exist in cluster %s", requiredStorageClass, cluster.Name))
	}

	if err := appManager.create(ctx, cluster, ZCloudNamespace, app); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("create monitor application failed %s", err.Error()))
	}

	go ensureApplicationSucceedOrDie(db, cluster, tableName, app.Name)
	return nil
}

func checkSysApplicationExist(db storage.DB, tableName, chartName string) (bool, error) {
	app, err := getApplicationFromDBByChartName(db, tableName, chartName)
	return app != nil, err
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

	app, err := getApplicationFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), monitorChartName)
	if err != nil || app == nil {
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
	return deleteApplicationByChartName(m.clusters.GetDB(), cluster, monitorChartName)
}

func deleteApplicationByChartName(db storage.DB, cluster *zke.Cluster, chartName string) *resterr.APIError {
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}
	app, err := getApplicationFromDBByChartName(db, storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), chartName)
	if err != nil || app == nil {
		return resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("%s doesn't exist", chartName))
	}

	appName := app.Name
	if err := deleteApplication(db, cluster.KubeClient, cluster.Name, ZCloudNamespace, appName, true); err != nil {
		if err == storage.ErrNotFoundResource {
			return resterr.NewAPIError(resterr.NotFound,
				fmt.Sprintf("application %s with namespace %s doesn't exist", appName, ZCloudNamespace))
		} else {
			return resterr.NewAPIError(resterr.ServerError,
				fmt.Sprintf("delete application %s failed: %s", appName, err.Error()))
		}
	}
	return nil
}

func genMonitorApplication(cluster *zke.Cluster, m *types.Monitor, clusterName string) (*types.Application, error) {
	config, err := genMonitorApplicationConfig(cluster, m, clusterName)
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

func genMonitorApplicationConfig(cluster *zke.Cluster, m *types.Monitor, clusterName string) ([]byte, error) {
	if len(m.IngressDomain) == 0 {
		edgeIPs := cluster.GetNodeIpsByRole(types.RoleEdge)
		if len(edgeIPs) == 0 {
			return nil, fmt.Errorf("can not find edge node for this cluster")
		}
		m.IngressDomain = monitorAppNamePrefix + "-" + ZCloudNamespace + "-" + clusterName + "." + edgeIPs[0] + "." + ZcloudDynamicaDomainPrefix
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
				StorageSize:    strconv.Itoa(m.StorageSize) + "Gi",
				Retention:      strconv.Itoa(m.PrometheusRetention) + "d",
				ScrapeInterval: strconv.Itoa(m.ScrapeInterval) + "s",
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
		IngressDomain: p.Grafana.Ingress.Hosts,
		StorageClass:  p.Prometheus.PrometheusSpec.StorageClass,
		RedirectUrl:   "http://" + p.Grafana.Ingress.Hosts,
		Status:        app.Status,
	}
	m.SetID(monitorAppNamePrefix)
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

func ensureApplicationSucceedOrDie(db storage.DB, cluster *zke.Cluster, tableName, appName string) {
	for i := 0; i < sysApplicationCheckTimes; i++ {
		sysApp, err := getApplicationFromDB(db, tableName, appName, true)
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
			if err := deleteApplication(db, cluster.KubeClient, cluster.Name, ZCloudNamespace, appName, true); err != nil {
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

func getApplicationFromDBByChartName(db storage.DB, tableName, chartName string) (*types.Application, error) {
	apps, err := getApplicationsFromDB(db, tableName)
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

func genAppNameIfDuplicate(db storage.DB, tableName, appName, appNamePrefex string) string {
	for {
		duplicateApp, _ := getApplicationFromDB(db, tableName, appName, true)
		if duplicateApp != nil {
			appName = appNamePrefex + "-" + genRandomStr(12)
		} else {
			break
		}
	}

	return appName
}
