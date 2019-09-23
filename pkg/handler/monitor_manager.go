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
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
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
	monitor := ctx.Resource.(*types.Monitor)
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	existApps, err := getApplicationsFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), monitorChartName)

	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get monitor application from db failed %s", err.Error()))
	}
	if len(existApps) > 0 {
		return nil, resterr.NewAPIError(resterr.DuplicateResource, fmt.Sprintf("cluster %s monitor has exist", cluster.Name))
	}

	// check the storage class exist
	requiredStorageClass := monitorStorageClass
	if len(monitor.StorageClass) > 0 {
		requiredStorageClass = monitor.StorageClass
	}
	if !isStorageClassExist(cluster.KubeClient, requiredStorageClass) {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("%s storageclass does't exist in cluster %s", requiredStorageClass, cluster.Name))
	}

	app, err := genMonitorApplication(cluster.KubeClient, monitor, cluster.Name)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, err.Error())
	}

	// check duplicate application resource, if exist, gen a new name for monitor app
	for {
		duplicateApp, _ := getSysApplicationFromDB(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), app.Name)
		if duplicateApp != nil {
			app.Name = monitorAppNamePrefix + "-" + genRandomStr(12)
		} else {
			break
		}
	}

	app.SetID(app.Name)
	if err := m.apps.create(ctx, cluster, ZCloudNamespace, app); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("create monitor application failed %s", err.Error()))
	}

	// make sure the monitor application is succeed, if it failed will delete this monitor application
	go ensureApplicationSucceedOrDie(ctx, m.clusters.GetDB(), cluster, storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), app.Name)

	monitor.Status = types.AppStatusCreate
	monitor.SetID(monitorAppNamePrefix)
	monitor.SetCreationTimestamp(time.Now())
	return monitor, nil
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

func (m *MonitorManager) Delete(ctx *restresource.Context) *resterr.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	apps, err := getApplicationsFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), monitorChartName)
	if err != nil || len(apps) == 0 {
		return resterr.NewAPIError(resterr.NotFound, "monitor doesn't exist")
	}
	appName := apps[0].Name

	app, err := updateApplicationStatusFromDB(m.clusters.GetDB(), getCurrentUser(ctx), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), appName, types.AppStatusDelete)
	if err != nil {
		if err == storage.ErrNotFoundResource {
			return resterr.NewAPIError(resterr.NotFound,
				fmt.Sprintf("application %s with namespace %s doesn't exist", appName, ZCloudNamespace))
		} else {
			return resterr.NewAPIError(resterr.ServerError,
				fmt.Sprintf("delete application %s failed: %s", appName, err.Error()))
		}
	}

	go deleteApplication(m.clusters.GetDB(), cluster.KubeClient, storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), ZCloudNamespace, app)
	return nil
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

func genMonitorFromApp(ctx *restresource.Context, cluster string, app *types.Application) (*types.Monitor, error) {
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

func ensureApplicationSucceedOrDie(ctx *restresource.Context, db storage.DB, cluster *zke.Cluster, tableName, appName string) {
	for i := 0; i < sysApplicationCheckTimes; i++ {
		sysApp, err := getSysApplicationFromDB(db, tableName, appName)
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
