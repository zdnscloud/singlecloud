package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"helm.sh/helm/pkg/chart/loader"
	"helm.sh/helm/pkg/chartutil"
	"helm.sh/helm/pkg/engine"

	appv1beta1 "github.com/zdnscloud/application-operator/pkg/apis/app/v1beta1"
	appctrl "github.com/zdnscloud/application-operator/pkg/controller"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
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
	crdCheckTimes       = 20
	crdCheckInterval    = 5 * time.Second
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
	if err := createApplication(ctx, cluster, namespace, m.chartDir, app, false); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource,
				fmt.Sprintf("duplicate application %s with namespace %s", app.Name, namespace))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create application failed %s", err.Error()))
		}
	}

	app.SetID(app.Name)
	return app, nil
}

func createApplication(ctx *resource.Context, cluster *zke.Cluster, namespace, chartDir string, app *types.Application, isSystemChart bool) error {
	if hasNamespace(cluster.GetKubeClient(), namespace) == false {
		return fmt.Errorf("namespace %s is not found", namespace)
	}

	if _, exists, err := getApplicationIfExists(cluster.GetKubeClient(), namespace, app.Name, isSystemChart); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("duplicate application %s with namespace %s for chart %s", app.Name, namespace, app.ChartName)
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

	if err := preInstallChartCRDs(cluster.GetKubeClient(), crdManifests); err != nil {
		return fmt.Errorf("create application %s crds failed: %s", app.Name, err.Error())
	}

	urls := strings.SplitAfterN(ctx.Request.URL.Path, fmt.Sprintf("/clusters/%s/namespaces/", cluster.Name), 2)
	return cluster.GetKubeClient().Create(context.TODO(), &appv1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: namespace,
			Annotations: map[string]string{
				appctrl.ZcloudAppRequestUrlPrefix: urls[0],
				AnnKeyForAppConfigs:               string(app.Configs),
			},
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
		},
	})
}

func getApplicationIfExists(cli client.Client, namespace, name string, isSystemChart bool) (*appv1beta1.Application, bool, error) {
	if app, err := getApplication(cli, namespace, name, isSystemChart); err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, false, fmt.Errorf("get app %s with namespace %s failed: %s", name, namespace, err.Error())
		}
		return nil, false, nil
	} else {
		return app, true, nil
	}

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

func (m *ApplicationManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		log.Warnf("no found cluster when list applications info")
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sAppCRDs, err := getApplications(cluster.GetKubeClient(), namespace)
	if err != nil {
		log.Warnf("list applications info failed:%s", err.Error())
		return nil
	}

	var apps types.Applications
	for _, k8sAppCRD := range k8sAppCRDs.Items {
		if k8sAppCRD.Spec.OwnerChart.SystemChart {
			continue
		}

		apps = append(apps, k8sAppCRDToScApp(&k8sAppCRD))
	}

	sort.Sort(apps)
	return apps
}

func getApplications(cli client.Client, namespace string) (*appv1beta1.ApplicationList, error) {
	apps := appv1beta1.ApplicationList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &apps)
	return &apps, err
}

func (m *ApplicationManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		log.Warnf("no found cluster when get application info")
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	appName := ctx.Resource.GetID()
	k8sAppCRD, exists, err := getApplicationIfExists(cluster.GetKubeClient(), namespace, appName, false)
	if err != nil {
		log.Warnf("get application %s info failed:%s", appName, err.Error())
	} else if exists {
		return k8sAppCRDToScApp(k8sAppCRD)
	}

	return nil
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

func k8sAppCRDToScApp(k8sAppCRD *appv1beta1.Application) *types.Application {
	var appResources types.AppResources
	for _, r := range k8sAppCRD.Status.AppResources {
		appResources = append(appResources, types.AppResource{
			Namespace:         r.Namespace,
			Name:              r.Name,
			Type:              string(r.Type),
			Link:              r.Link,
			Replicas:          r.Replicas,
			ReadyReplicas:     r.ReadyReplicas,
			Exists:            r.Exists,
			CreationTimestamp: resource.ISOTime(r.CreationTimestamp.Time),
		})
	}

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
		return resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("delete application %s with namespace %s failed: %s", appName, namespace, err.Error()))
	}

	return nil
}

func deleteApplication(cli client.Client, namespace, name string, isSystemChart bool) error {
	app, exists, err := getApplicationIfExists(cli, namespace, name, isSystemChart)
	if err != nil {
		return fmt.Errorf("get application %s with namespace %s failed: %s", name, namespace, err.Error())
	} else if exists == false {
		return fmt.Errorf("application %s with namespace %s doesn't exist", name, namespace)
	}

	return cli.Delete(context.TODO(), app)
}

func isCRDReady(crd apiextv1beta1.CustomResourceDefinition) bool {
	for _, cond := range crd.Status.Conditions {
		switch cond.Type {
		case apiextv1beta1.Established:
			if cond.Status == apiextv1beta1.ConditionTrue {
				return true
			}
		case apiextv1beta1.NamesAccepted:
			if cond.Status == apiextv1beta1.ConditionFalse {
				return true
			}
		}
	}
	return false
}

func isCRDsReady(requiredCRDs []*apiextv1beta1.CustomResourceDefinition, allCRDs apiextv1beta1.CustomResourceDefinitionList) bool {
	for _, required := range requiredCRDs {
		ready := false
		for _, crd := range allCRDs.Items {
			if crd.Name == required.Name {
				if isCRDReady(crd) {
					ready = true
				}
				break
			}
		}
		if !ready {
			return false
		}
	}
	return true
}

func waitCRDsReady(client client.Client, requiredCRDs []*apiextv1beta1.CustomResourceDefinition) bool {
	var allCRDs apiextv1beta1.CustomResourceDefinitionList
	for i := 0; i < crdCheckTimes; i++ {
		if err := client.List(context.TODO(), nil, &allCRDs); err == nil {
			if isCRDsReady(requiredCRDs, allCRDs) {
				return true
			}
		}
		time.Sleep(crdCheckInterval)
	}
	return false
}

func preInstallChartCRDs(cli client.Client, manifests []appv1beta1.Manifest) error {
	if len(manifests) == 0 {
		return nil
	}

	var crds []*apiextv1beta1.CustomResourceDefinition
	for _, manifest := range manifests {
		if err := helper.MapOnRuntimeObject(manifest.Content, func(ctx context.Context, obj runtime.Object) error {
			crd, ok := obj.(*apiextv1beta1.CustomResourceDefinition)
			if !ok {
				return fmt.Errorf("runtime object isn't k8s crd object")
			}
			crds = append(crds, crd)

			if err := cli.Create(ctx, obj); err != nil {
				if apierrors.IsAlreadyExists(err) {
					log.Infof("ignore already exist crd %s", crd.Name)
					return nil
				}
				return fmt.Errorf("create crd with file %s failed: %s", manifest.File, err.Error())
			}
			return nil
		}); err != nil {
			return err
		}
	}

	if !waitCRDsReady(cli, crds) {
		return fmt.Errorf("preinstall chart crds timeout")
	}
	return nil
}
