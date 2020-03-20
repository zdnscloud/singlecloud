package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"

	appv1beta1 "github.com/zdnscloud/application-operator/pkg/apis/app/v1beta1"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	monitorAppName      = "monitor"
	monitorChartName    = "prometheus"
	monitorChartVersion = "6.4.3"

	ZcloudDynamicaDomainPrefix  = "zc.zdns.cn"
	sysApplicationCheckInterval = time.Second * 5
	sysApplicationCheckTimes    = 30
)

type MonitorManager struct {
	clusters *ClusterManager
	chartDir string
}

func newMonitorManager(clusterMgr *ClusterManager, chartDir string) *MonitorManager {
	return &MonitorManager{
		clusters: clusterMgr,
		chartDir: chartDir,
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
		return nil, resterr.NewAPIError(types.ConnectClusterFailed, err.Error())
	}

	if err := createSysApplication(ctx, cluster, app, m.chartDir, monitorChartName, monitorAppName, monitor.StorageClass); err != nil {
		return nil, err
	}

	monitor.SetID(monitorAppName)
	return monitor, nil
}

func createSysApplication(ctx *restresource.Context, cluster *zke.Cluster, app *types.Application, chartDir, chartName, appName, storageClass string) *resterr.APIError {
	if !isStorageClassExist(cluster.GetKubeClient(), storageClass) {
		return resterr.NewAPIError(resterr.PermissionDenied,
			fmt.Sprintf("%s storageclass does't exist in cluster %s", storageClass, cluster.Name))
	}

	app.SetID(appName)
	if err := createApplication(ctx, cluster, ZCloudNamespace, chartDir, app, true); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("create %s application failed %s", chartName, err.Error()))
	}

	go ensureApplicationSucceedOrDie(cluster.GetKubeClient(), appName)
	return nil
}

func (m *MonitorManager) List(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	monitor, err := m.get(cluster.GetKubeClient())
	if err != nil {
		if err.ErrorCode == resterr.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return []*types.Monitor{monitor.(*types.Monitor)}, nil
}

func (m *MonitorManager) Get(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	id := ctx.Resource.GetID()
	if id != monitorAppName {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("monitor %s doesn't exist", id))
	}
	return m.get(cluster.GetKubeClient())
}

func (m *MonitorManager) get(cli client.Client) (restresource.Resource, *resterr.APIError) {
	k8sAppCRD, err := getApplication(cli, ZCloudNamespace, monitorAppName, true)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, "monitor doesn't exist")
		}
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get application monitor by chart name %s failed %s", monitorChartName, err.Error()))
	}

	monitor, err := genRetrunMonitorFromApplication(k8sAppCRD)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("parse k8s app crd to monitor failed: %s", err.Error()))
	}
	return monitor, nil
}

func (m *MonitorManager) Delete(ctx *restresource.Context) *resterr.APIError {
	if ctx.Resource.GetID() != monitorAppName {
		return resterr.NewAPIError(resterr.NotFound, "monitor doesn't exist")
	}

	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	if err := deleteApplication(cluster.GetKubeClient(), ZCloudNamespace, monitorAppName, true); err != nil {
		if apierrors.IsNotFound(err) {
			return resterr.NewAPIError(resterr.NotFound, "monitor doesn't exist")
		}
		return resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("delete application %s failed: %s", monitorAppName, err.Error()))
	}

	return nil
}

func genMonitorApplication(cluster *zke.Cluster, m *types.Monitor) (*types.Application, error) {
	config, err := genMonitorApplicationConfig(cluster, m)
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

func genMonitorApplicationConfig(cluster *zke.Cluster, m *types.Monitor) ([]byte, error) {
	domain, err := genIngressDomain(cluster, m.IngressDomain, monitorAppName)
	if err != nil {
		return nil, err
	}
	m.IngressDomain = domain

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

func genIngressDomain(cluster *zke.Cluster, ingressDomain, appName string) (string, error) {
	if len(ingressDomain) != 0 {
		return ingressDomain, nil
	}

	edgeIP := getRandomEdgeNodeAddress(cluster)
	if len(edgeIP) == 0 {
		return "", fmt.Errorf("can not find edge node for this cluster")
	}
	return appName + "-" + ZCloudNamespace + "-" + cluster.Name + "." + edgeIP + "." + ZcloudDynamicaDomainPrefix, nil
}

func genRetrunMonitorFromApplication(app *appv1beta1.Application) (*types.Monitor, error) {
	p := charts.Prometheus{}
	if err := getAppConfigsFromAnnotations(app, &p); err != nil {
		return nil, err
	}

	m := types.Monitor{
		IngressDomain:       p.Grafana.Ingress.Hosts,
		StorageClass:        p.Prometheus.PrometheusSpec.StorageClass,
		StorageSize:         p.Prometheus.PrometheusSpec.StorageSize,
		PrometheusRetention: p.Prometheus.PrometheusSpec.Retention,
		ScrapeInterval:      p.Prometheus.PrometheusSpec.ScrapeInterval,
		RedirectUrl:         "http://" + p.Grafana.Ingress.Hosts,
		Status:              string(app.Status.State),
	}
	m.SetID(monitorAppName)
	m.SetCreationTimestamp(app.CreationTimestamp.Time)
	if app.GetDeletionTimestamp() != nil {
		m.SetDeletionTimestamp(app.DeletionTimestamp.Time)
		m.Status = appStatusDelete
	}
	return &m, nil
}

func getAppConfigsFromAnnotations(app *appv1beta1.Application, appConfigs interface{}) error {
	if configs, ok := app.Annotations[AnnKeyForAppConfigs]; ok {
		if err := json.Unmarshal([]byte(configs), appConfigs); err != nil {
			return fmt.Errorf("unmarshal app.configs annotation for app %s with namespace %s failed: %s",
				app.Name, app.Namespace, err.Error())
		}
		return nil
	}

	return fmt.Errorf("no found app.configs annotation for app %s with namespace %s", app.Name, app.Namespace)
}

func ensureApplicationSucceedOrDie(cli client.Client, appName string) {
	for i := 0; i < sysApplicationCheckTimes; i++ {
		app, err := getApplication(cli, ZCloudNamespace, appName, true)
		if err != nil {
			if apierrors.IsNotFound(err) == false {
				log.Warnf("get system application %s failed %s", appName, err.Error())
				return
			} else {
				time.Sleep(sysApplicationCheckInterval)
				continue
			}
		}

		switch app.Status.State {
		case appv1beta1.ApplicationStatusStateFailed:
			if err := deleteApplication(cli, ZCloudNamespace, appName, true); err != nil {
				log.Warnf("delete system application %s failed %s", appName, err.Error())
				return
			}
		case appv1beta1.ApplicationStatusStateSucceed:
			log.Infof("create system application %s succeed", appName)
			return
		}
	}
}
