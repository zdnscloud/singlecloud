package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"helm.sh/helm/pkg/chart/loader"
	"helm.sh/helm/pkg/chartutil"
	"helm.sh/helm/pkg/engine"

	appv1beta1 "github.com/zdnscloud/application-operator/pkg/apis/app/v1beta1"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	restutil "github.com/zdnscloud/gorest/util"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

var (
	DefaultCapabilities = &chartutil.Capabilities{
		APIVersions: chartutil.DefaultVersionSet,
	}
)

const (
	notesFileSuffix     = "NOTES.txt"
	appStatusDelete     = "delete"
	AnnKeyForAppConfigs = "app.configs"
)

type ApplicationManager struct {
	clusters *ClusterManager
	chartDir string
}

func newApplicationManager(clusters *ClusterManager, chartDir string) *ApplicationManager {
	return &ApplicationManager{
		clusters: clusters,
		chartDir: chartDir,
	}
}

func (m *ApplicationManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	app := ctx.Resource.(*types.Application)
	chart, err := getChart(m.chartDir, app.ChartName, false)
	if err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("get chart %s failed %s", app.ChartName, err.Error()))
	}

	if err := createApplication(ctx, cluster, namespace, chart.Dir, app, false); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate application %s", app.Name))
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create application failed %s", err.Error()))
	}

	app.SetID(app.Name)
	return app, nil
}

func createApplication(ctx *resource.Context, cluster *zke.Cluster, namespace, chartDir string, app *types.Application, isSystemChart bool) error {
	if hasNamespace(cluster.GetKubeClient(), namespace) == false {
		return fmt.Errorf("namespace %s is not found", namespace)
	}

	if _, err := getApplication(cluster.GetKubeClient(), namespace, app.Name, isSystemChart); err != nil {
		if apierrors.IsNotFound(err) == false {
			return fmt.Errorf("get application %s for chart %s failed: %s", app.Name, app.ChartName, err.Error())
		}
	} else {
		return fmt.Errorf("duplicate application %s", app.Name)
	}

	chartVersionDir := path.Join(chartDir, app.ChartName, app.ChartVersion)
	info, err := getChartInfo(chartVersionDir)
	if err != nil {
		return fmt.Errorf("load chart %s with version %s info failed: %s", app.ChartName, app.ChartVersion, err.Error())
	}

	systemChart := slice.SliceIndex(info.Keywords, KeywordZcloudSystem) != -1
	if systemChart != isSystemChart {
		return fmt.Errorf("can`t use application interface create systemchart application with chart %s", app.ChartName)
	}

	if clusterVersion, err := cluster.GetKubeClient().ServerVersion(); err != nil {
		return fmt.Errorf("get cluster %s version failed: %s", cluster.Name, err.Error())
	} else {
		DefaultCapabilities.KubeVersion.Version = clusterVersion.GitVersion
		DefaultCapabilities.KubeVersion.Major = clusterVersion.Major
		DefaultCapabilities.KubeVersion.Minor = clusterVersion.Minor
	}

	configs, err := parseChartConfigs(chartVersionDir, app.Configs)
	if err != nil {
		return fmt.Errorf("parse chart %s with version %s configs failed: %s", app.ChartName, app.ChartVersion, err.Error())
	}

	manifests, crdManifests, err := loadChartFiles(namespace, chartVersionDir, app.Name, configs, DefaultCapabilities)
	if err != nil {
		return fmt.Errorf("load chart %s with version %s files failed: %s", app.ChartName, app.ChartVersion, err.Error())
	}

	k8sAppCRD := &appv1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: namespace,
		},
		Spec: appv1beta1.ApplicationSpec{
			OwnerChart: appv1beta1.ChartInfo{
				Name:        app.ChartName,
				Version:     app.ChartVersion,
				Icon:        genChartIcon(iconPrefixForReturn, app.ChartName),
				SystemChart: systemChart,
			},
			InjectServiceMesh: app.InjectServiceMesh,
			CreatedByAdmin:    isAdmin(getCurrentUser(ctx)),
			Manifests:         manifests,
			CRDManifests:      crdManifests,
		},
	}

	if len(app.Configs) != 0 {
		k8sAppCRD.Annotations = map[string]string{AnnKeyForAppConfigs: string(app.Configs)}
	}
	return cluster.GetKubeClient().Create(context.TODO(), k8sAppCRD)
}

func parseChartConfigs(chartVersionDir string, configRaw json.RawMessage) (map[string]interface{}, error) {
	configs := make(map[string]interface{})
	if configRaw == nil {
		return configs, nil
	}

	if err := json.Unmarshal(configRaw, &configs); err != nil {
		return nil, fmt.Errorf("unmarshal chart configs failed: %v", err.Error())
	}

	if err := charts.CheckConfigs(chartVersionDir, configs); err != nil {
		return nil, err
	}

	return configs, nil
}

func loadChartFiles(namespace, chartVersionDir, appName string, configs map[string]interface{}, caps *chartutil.Capabilities) ([]appv1beta1.Manifest, []appv1beta1.Manifest, error) {
	chartRequested, err := loader.Load(chartVersionDir)
	if err != nil {
		return nil, nil, err
	}

	options := chartutil.ReleaseOptions{
		Name:      appName,
		Namespace: namespace,
		IsInstall: true,
	}
	valuesToRender, err := chartutil.ToRenderValues(chartRequested, configs, options, caps)
	if err != nil {
		return nil, nil, err
	}
	if rel, ok := valuesToRender["Release"].(map[string]interface{}); ok {
		rel["Service"] = "zcloud"
	}

	files, err := engine.Render(chartRequested, valuesToRender)
	if err != nil {
		return nil, nil, err
	}

	var manifests []appv1beta1.Manifest
	for fileName, content := range files {
		if strings.HasSuffix(fileName, notesFileSuffix) {
			delete(files, fileName)
		} else {
			manifests = append(manifests, appv1beta1.Manifest{
				File:    fileName,
				Content: content,
			})
		}
	}

	var crdManifests []appv1beta1.Manifest
	for _, crdFile := range chartRequested.CRDs() {
		crdManifests = append(crdManifests, appv1beta1.Manifest{
			File:    crdFile.Name,
			Content: string(crdFile.Data),
		})
	}

	return manifests, crdManifests, nil
}

func (m *ApplicationManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sAppCRDs, err := getApplications(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no applications found")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list applications failed %s", err.Error()))
	}

	urlPrefix := getRequestUrlPrefix(ctx.Request.URL.Path, cluster.Name)
	var apps types.Applications
	for _, k8sAppCRD := range k8sAppCRDs.Items {
		if k8sAppCRD.Spec.OwnerChart.SystemChart {
			continue
		}

		apps = append(apps, k8sAppCRDToScApp(&k8sAppCRD, urlPrefix))
	}

	sort.Sort(apps)
	return apps, nil
}

func getApplications(cli client.Client, namespace string) (*appv1beta1.ApplicationList, error) {
	apps := appv1beta1.ApplicationList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &apps)
	return &apps, err
}

func getRequestUrlPrefix(reqUrlPath, clusterName string) string {
	return strings.SplitAfterN(reqUrlPath, fmt.Sprintf("/clusters/%s/namespaces/", clusterName), 2)[0]
}

func (m *ApplicationManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	appName := ctx.Resource.GetID()
	k8sAppCRD, err := getApplication(cluster.GetKubeClient(), namespace, appName, false)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found application %s", appName))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get application %s failed:%s", appName, err.Error()))
	}

	return k8sAppCRDToScApp(k8sAppCRD, getRequestUrlPrefix(ctx.Request.URL.Path, cluster.Name)), nil
}

func getApplication(cli client.Client, namespace, name string, isSystemChart bool) (*appv1beta1.Application, error) {
	app := appv1beta1.Application{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &app); err != nil {
		return nil, err
	}

	if app.Spec.OwnerChart.SystemChart != isSystemChart {
		return nil, fmt.Errorf("user no authority access application %s with namespace %s", name, namespace)
	}

	return &app, nil
}

func k8sAppCRDToScApp(k8sAppCRD *appv1beta1.Application, urlPrefix string) *types.Application {
	var appResources types.AppResources
	for _, r := range k8sAppCRD.Status.AppResources {
		appResource := types.AppResource{
			Namespace:         r.Namespace,
			Name:              r.Name,
			Type:              string(r.Type),
			Replicas:          r.Replicas,
			ReadyReplicas:     r.ReadyReplicas,
			Exists:            r.Exists,
			CreationTimestamp: resource.ISOTime(r.CreationTimestamp.Time),
		}
		if r.Exists {
			appResource.Link = path.Join(urlPrefix, r.Namespace, restutil.GuessPluralName(string(r.Type)), r.Name)
		}

		appResources = append(appResources, appResource)
	}

	sort.Sort(appResources)
	app := &types.Application{
		Name:               k8sAppCRD.Name,
		ChartName:          k8sAppCRD.Spec.OwnerChart.Name,
		ChartVersion:       k8sAppCRD.Spec.OwnerChart.Version,
		ChartIcon:          k8sAppCRD.Spec.OwnerChart.Icon,
		InjectServiceMesh:  k8sAppCRD.Spec.InjectServiceMesh,
		Status:             string(k8sAppCRD.Status.State),
		WorkloadCount:      k8sAppCRD.Status.WorkloadCount,
		ReadyWorkloadCount: k8sAppCRD.Status.ReadyWorkloadCount,
		AppResources:       appResources,
	}
	app.SetID(app.Name)
	app.SetCreationTimestamp(k8sAppCRD.CreationTimestamp.Time)
	if k8sAppCRD.GetDeletionTimestamp() != nil {
		app.SetDeletionTimestamp(k8sAppCRD.DeletionTimestamp.Time)
		app.Status = appStatusDelete
	}
	return app
}

func (m *ApplicationManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	appName := ctx.Resource.GetID()
	if err := deleteApplication(cluster.GetKubeClient(), namespace, appName, false); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found application %s", appName))
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete application %s failed: %s", appName, err.Error()))
	}

	return nil
}

func deleteApplication(cli client.Client, namespace, name string, isSystemChart bool) error {
	if _, err := getApplication(cli, namespace, name, isSystemChart); err != nil {
		return err
	}

	return cli.Delete(context.TODO(), &appv1beta1.Application{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
}
