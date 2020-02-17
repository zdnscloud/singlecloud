package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"helm.sh/helm/pkg/chart/loader"
	"helm.sh/helm/pkg/chartutil"
	"helm.sh/helm/pkg/engine"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/gorest/resource/schema/resourcefield"
	restutil "github.com/zdnscloud/gorest/util"
	"github.com/zdnscloud/kvzoo"
	"github.com/zdnscloud/singlecloud/pkg/alarm"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/db"
	eb "github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

var (
	DefaultCapabilities = &chartutil.Capabilities{
		APIVersions: chartutil.DefaultVersionSet,
	}

	SupportWorkloadTypes = []string{types.ResourceTypeDeployment, types.ResourceTypeDaemonSet, types.ResourceTypeStatefulSet}
	SupportResourceTypes = append(SupportWorkloadTypes, types.ResourceTypeCronJob, types.ResourceTypeJob, types.ResourceTypeConfigMap, types.ResourceTypeSecret, types.ResourceTypeService, types.ResourceTypeIngress)
)

const (
	notesFileSuffix  = "NOTES.txt"
	ApplicationTable = "application"

	crdCheckTimes        = 20
	crdCheckInterval     = 5 * time.Second
	createFailedReason   = "create failed"
	updateDBFailedReason = "update database failed"
	deleteFailedReason   = "delete failed"
)

type ApplicationManager struct {
	clusters       *ClusterManager
	chartDir       string
	clusterEventCh <-chan interface{}
}

type chartConfig struct {
	structVal reflect.Value
	fields    resourcefield.ResourceField
}

func newApplicationManager(clusters *ClusterManager, chartDir string) *ApplicationManager {
	m := &ApplicationManager{
		clusters:       clusters,
		chartDir:       chartDir,
		clusterEventCh: eb.SubscribeResourceEvent(types.Cluster{}),
	}
	go m.eventLoop()
	return m
}

func (m *ApplicationManager) eventLoop() {
	for {
		event := <-m.clusterEventCh
		switch e := event.(type) {
		case eb.ResourceDeleteEvent:
			tn, _ := kvzoo.TableNameFromSegments(ApplicationTable, e.Resource.GetID())
			if err := db.GetGlobalDB().DeleteTable(tn); err != nil {
				log.Warnf("delete /application/cluster %s table failed: %s", e.Resource.GetID(), err.Error())
			}
		}
	}
}

func (m *ApplicationManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	app := ctx.Resource.(*types.Application)
	app.SetID(app.Name)
	if err := m.createApplication(ctx, cluster, namespace, app, false); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource,
				fmt.Sprintf("duplicate chart %s with name %s", app.ChartName, app.Name))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create application failed %s", err.Error()))
		}
	}

	return genReturnApplication(app), nil
}

func (m *ApplicationManager) createApplication(ctx *resource.Context, cluster *zke.Cluster, namespace string, app *types.Application, isSystemChart bool) error {
	if hasNamespace(cluster.KubeClient, namespace) == false {
		return fmt.Errorf("namespace %s is not found", namespace)
	}

	chartPath := path.Join(m.chartDir, app.ChartName, app.ChartVersion)
	info, err := getChartInfo(chartPath)
	if err != nil {
		return fmt.Errorf("load chart %s with version %s info failed: %s", app.ChartName, app.ChartVersion, err.Error())
	}

	if app.SystemChart = slice.SliceIndex(info.Keywords, KeywordZcloudSystem) != -1; app.SystemChart != isSystemChart {
		return fmt.Errorf("can`t use application interface create systemchart application with chart %s", app.ChartName)
	}

	if clusterVersion, err := cluster.KubeClient.ServerVersion(); err != nil {
		return fmt.Errorf("get cluster %s version failed: %s", cluster.Name, err.Error())
	} else {
		DefaultCapabilities.KubeVersion.Version = clusterVersion.GitVersion
		DefaultCapabilities.KubeVersion.Major = clusterVersion.Major
		DefaultCapabilities.KubeVersion.Minor = clusterVersion.Minor
	}

	configs, err := parseChartConfigs(path.Join(m.chartDir, app.ChartName, app.ChartVersion), app.Configs)
	if err != nil {
		return fmt.Errorf("parse chart %s with version %s configs failed: %s", app.ChartName, app.ChartVersion, err.Error())
	}

	manifests, crdManifests, err := loadChartFiles(namespace, chartPath, app.Name, configs, DefaultCapabilities)
	if err != nil {
		return fmt.Errorf("load chart %s with version %s files failed: %s", app.ChartName, app.ChartVersion, err.Error())
	}

	app.Manifests = manifests
	app.Status = types.AppStatusCreate
	app.SetCreationTimestamp(time.Now())
	app.ChartIcon = genChartIcon(iconPrefixForReturn, app.ChartName)
	table, _, err := createOrGetAppTable(cluster.Name, namespace)
	if err != nil {
		return err
	}

	if err := addApplicationToDB(table, app); err != nil {
		return fmt.Errorf("add application %s to db failed: %s", app.Name, err.Error())
	}

	urls := strings.SplitAfterN(ctx.Request.URL.Path, fmt.Sprintf("/clusters/%s/namespaces/", cluster.Name), 2)
	go func() {
		if err := asyncCreateApplication(table, cluster.KubeClient, isAdmin(getCurrentUser(ctx)), namespace,
			urls[0], app, crdManifests); err != nil {
			log.Warnf("create application failed: %s", err.Error())
			publishApplicationEvent(cluster.Name, namespace, app.Name, createFailedReason, err.Error())
			updateAppStatusToFailed(table, cluster.Name, namespace, app)
		}
	}()
	return nil
}

func publishApplicationEvent(cluster, namespace, name, reason, msg string) {
	alarm.New().
		Cluster(cluster).
		Namespace(namespace).
		Kind("Application").
		Name(name).
		Reason(reason).
		Message(msg).
		Publish()
}

func createOrGetAppTable(clusterName, namespace string) (kvzoo.Table, kvzoo.TableName, error) {
	tn, _ := kvzoo.TableNameFromSegments(ApplicationTable, clusterName, namespace)
	table, err := db.GetGlobalDB().CreateOrGetTable(tn)
	if err != nil {
		return nil, tn, fmt.Errorf("create or get table %s failed: %s", tn, err.Error())
	}

	return table, tn, nil
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

func loadChartFiles(namespace, chartPath, appName string, configs map[string]interface{}, caps *chartutil.Capabilities) ([]types.Manifest, []types.Manifest, error) {
	chartRequested, err := loader.Load(chartPath)
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

	var manifests []types.Manifest
	for fileName, content := range files {
		if strings.HasSuffix(fileName, notesFileSuffix) {
			delete(files, fileName)
		} else {
			manifests = append(manifests, types.Manifest{
				File:    fileName,
				Content: content,
			})
		}
	}

	var crdManifests []types.Manifest
	for _, crdFile := range chartRequested.CRDs() {
		crdManifests = append(crdManifests, types.Manifest{
			File:    crdFile.Name,
			Content: string(crdFile.Data),
		})
	}

	return manifests, crdManifests, nil
}

func addApplicationToDB(table kvzoo.Table, app *types.Application) error {
	return addOrUpdateApplicationToDB(table, app, true)
}

func updateApplicationToDB(table kvzoo.Table, app *types.Application) error {
	return addOrUpdateApplicationToDB(table, app, false)
}

func addOrUpdateApplicationToDB(table kvzoo.Table, app *types.Application, isCreate bool) error {
	value, err := json.Marshal(app)
	if err != nil {
		return fmt.Errorf("marshal application %s failed: %s", app.Name, err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %s", err.Error())
	}

	defer tx.Rollback()
	if isCreate {
		err = tx.Add(app.GetID(), value)
	} else {
		err = tx.Update(app.GetID(), value)
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}

func asyncCreateApplication(table kvzoo.Table, cli client.Client, isAdmin bool, namespace, urlPrefix string, app *types.Application, crdManifests []types.Manifest) error {
	if err := preInstallChartCRDs(cli, crdManifests); err != nil {
		return fmt.Errorf("create application %s crds failed: %s", app.Name, err.Error())
	}

	if err := createAppResources(cli, isAdmin, namespace, urlPrefix, app); err != nil {
		return fmt.Errorf("create application %s resources failed: %s", app.Name, err.Error())
	}

	app.Status = types.AppStatusSucceed
	if err := updateApplicationToDB(table, app); err != nil {
		return fmt.Errorf("update application %s status to succeed failed: %s", app.Name, err.Error())
	}

	return nil
}

func updateAppStatusToFailed(table kvzoo.Table, clusterName, namespace string, app *types.Application) {
	app.Status = types.AppStatusFailed
	if err := updateApplicationToDB(table, app); err != nil {
		log.Warnf("update application %s status to failed get error: %s", app.Name, err.Error())
		publishApplicationEvent(clusterName, namespace, app.Name, updateDBFailedReason, err.Error())
	}
}

func createAppResources(cli client.Client, isAdmin bool, namespace, urlPrefix string, app *types.Application) error {
	for i, manifest := range app.Manifests {
		if err := helper.MapOnRuntimeObject(manifest.Content, func(ctx context.Context, obj runtime.Object) error {
			if obj == nil {
				return fmt.Errorf("cann`t unmarshal file %s to k8s runtime object\n", manifest.File)
			}

			gvk := obj.GetObjectKind().GroupVersionKind()
			metaObj, err := runtimeObjectToMetaObject(obj, namespace, isAdmin)
			if err != nil {
				return fmt.Errorf("runtime object to meta object with chart file %s failed: %s", manifest.File, err.Error())
			}

			typ := strings.ToLower(gvk.Kind)
			injectServiceMeshToWorkload(typ, app, obj)
			if err := cli.Create(ctx, obj); err != nil {
				if apierrors.IsAlreadyExists(err) {
					app.Manifests[i].Duplicate = true
				}
				return fmt.Errorf("create resource with file %s failed: %s", manifest.File, err.Error())
			}

			if slice.SliceIndex(SupportResourceTypes, typ) != -1 {
				if slice.SliceIndex(SupportWorkloadTypes, typ) != -1 {
					app.WorkloadCount += 1
				}
				app.AppResources = append(app.AppResources, types.AppResource{
					Name: metaObj.GetName(),
					Type: typ,
					Link: path.Join(urlPrefix, metaObj.GetNamespace(), restutil.GuessPluralName(typ), metaObj.GetName()),
				})
			}
			return nil
		}); err != nil {
			return err
		}
	}

	sort.Sort(app.AppResources)
	return nil
}

func runtimeObjectToMetaObject(obj runtime.Object, namespace string, isAdmin bool) (metav1.Object, error) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	if metaObj.GetNamespace() != "" {
		if isAdmin == false {
			return nil, fmt.Errorf("chart file should not has namespace")
		}
	} else {
		metaObj.SetNamespace(namespace)
	}

	return metaObj, nil
}

func injectServiceMeshToWorkload(typ string, app *types.Application, obj runtime.Object) {
	if slice.SliceIndex(SupportWorkloadTypes, typ) == -1 || app.InjectServiceMesh == false {
		return
	}

	switch obj.(type) {
	case *appsv1.Deployment:
		deploy := obj.(*appsv1.Deployment)
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = make(map[string]string)
		}
		deploy.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	case *appsv1beta1.Deployment:
		deploy := obj.(*appsv1beta1.Deployment)
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = make(map[string]string)
		}
		deploy.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	case *appsv1beta2.Deployment:
		deploy := obj.(*appsv1beta2.Deployment)
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = make(map[string]string)
		}
		deploy.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	case *extv1beta1.Deployment:
		deploy := obj.(*extv1beta1.Deployment)
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = make(map[string]string)
		}
		deploy.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	case *appsv1.DaemonSet:
		ds := obj.(*appsv1.DaemonSet)
		if ds.Spec.Template.Annotations == nil {
			ds.Spec.Template.Annotations = make(map[string]string)
		}
		ds.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	case *appsv1beta2.DaemonSet:
		ds := obj.(*appsv1beta2.DaemonSet)
		if ds.Spec.Template.Annotations == nil {
			ds.Spec.Template.Annotations = make(map[string]string)
		}
		ds.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	case *extv1beta1.DaemonSet:
		ds := obj.(*extv1beta1.DaemonSet)
		if ds.Spec.Template.Annotations == nil {
			ds.Spec.Template.Annotations = make(map[string]string)
		}
		ds.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	case *appsv1.StatefulSet:
		sts := obj.(*appsv1.StatefulSet)
		if sts.Spec.Template.Annotations == nil {
			sts.Spec.Template.Annotations = make(map[string]string)
		}
		sts.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	case *appsv1beta1.StatefulSet:
		sts := obj.(*appsv1beta1.StatefulSet)
		if sts.Spec.Template.Annotations == nil {
			sts.Spec.Template.Annotations = make(map[string]string)
		}
		sts.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	case *appsv1beta2.StatefulSet:
		sts := obj.(*appsv1beta2.StatefulSet)
		if sts.Spec.Template.Annotations == nil {
			sts.Spec.Template.Annotations = make(map[string]string)
		}
		sts.Spec.Template.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	}
}

func (m *ApplicationManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	table, _, err := createOrGetAppTable(cluster.Name, namespace)
	if err != nil {
		return err
	}

	allApps, err := getApplicationsFromDB(table)
	if err != nil {
		log.Warnf("list applications failed %s", err.Error())
		return nil
	}

	var apps types.Applications
	for _, app := range allApps {
		if app.SystemChart == false {
			getAppResources(cluster.KubeClient, namespace, app)
			apps = append(apps, genReturnApplication(app))
		}
	}

	sort.Sort(apps)
	return apps
}

func getApplicationsFromDB(table kvzoo.Table) (types.Applications, error) {
	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()
	appValues, err := tx.List()
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var apps types.Applications
	for _, value := range appValues {
		if len(value) == 0 {
			continue
		}

		var app types.Application
		if err := json.Unmarshal(value, &app); err != nil {
			return nil, err
		}

		apps = append(apps, &app)
	}

	return apps, nil
}

func getAppResources(cli client.Client, namespace string, app *types.Application) {
	var resources types.AppResources
	for _, r := range app.AppResources {
		var err error
		var k8sResource interface{}
		switch r.Type {
		case types.ResourceTypeDeployment:
			k8sResource, err = getDeployment(cli, namespace, r.Name)
			if err == nil {
				r.Replicas = int(*k8sResource.(*appsv1.Deployment).Spec.Replicas)
				r.ReadyReplicas = int(k8sResource.(*appsv1.Deployment).Status.ReadyReplicas)
				r.CreationTimestamp = resource.ISOTime(k8sResource.(*appsv1.Deployment).CreationTimestamp.Time)
			}
		case types.ResourceTypeDaemonSet:
			k8sResource, err = getDaemonSet(cli, namespace, r.Name)
			if err == nil {
				r.Replicas = int(k8sResource.(*appsv1.DaemonSet).Status.DesiredNumberScheduled)
				r.ReadyReplicas = int(k8sResource.(*appsv1.DaemonSet).Status.NumberReady)
				r.CreationTimestamp = resource.ISOTime(k8sResource.(*appsv1.DaemonSet).CreationTimestamp.Time)
			}
		case types.ResourceTypeStatefulSet:
			k8sResource, err = getStatefulSet(cli, namespace, r.Name)
			if err == nil {
				r.Replicas = int(*k8sResource.(*appsv1.StatefulSet).Spec.Replicas)
				r.ReadyReplicas = int(k8sResource.(*appsv1.StatefulSet).Status.ReadyReplicas)
				r.CreationTimestamp = resource.ISOTime(k8sResource.(*appsv1.StatefulSet).CreationTimestamp.Time)
			}
		case types.ResourceTypeCronJob:
			k8sResource, err = getCronJob(cli, namespace, r.Name)
			if err == nil {
				r.CreationTimestamp = resource.ISOTime(k8sResource.(*batchv1beta1.CronJob).CreationTimestamp.Time)
			}
		case types.ResourceTypeJob:
			k8sResource, err = getJob(cli, namespace, r.Name)
			if err == nil {
				r.CreationTimestamp = resource.ISOTime(k8sResource.(*batchv1.Job).CreationTimestamp.Time)
			}
		case types.ResourceTypeConfigMap:
			k8sResource, err = getConfigMap(cli, namespace, r.Name)
			if err == nil {
				r.CreationTimestamp = resource.ISOTime(k8sResource.(*corev1.ConfigMap).CreationTimestamp.Time)
			}
		case types.ResourceTypeSecret:
			k8sResource, err = getSecret(cli, namespace, r.Name)
			if err == nil {
				r.CreationTimestamp = resource.ISOTime(k8sResource.(*corev1.Secret).CreationTimestamp.Time)
			}
		case types.ResourceTypeService:
			k8sResource, err = getService(cli, namespace, r.Name)
			if err == nil {
				r.CreationTimestamp = resource.ISOTime(k8sResource.(*corev1.Service).CreationTimestamp.Time)
			}
		case types.ResourceTypeIngress:
			k8sResource, err = getIngress(cli, namespace, r.Name)
			if err == nil {
				r.CreationTimestamp = resource.ISOTime(k8sResource.(*extv1beta1.Ingress).CreationTimestamp.Time)
			}
		}

		if err != nil {
			log.Warnf("get application %s resource %s/%s failed %s", app.Name, r.Type, r.Name, err.Error())
			r.Link = ""
		} else {
			r.Exists = true
			if r.ReadyReplicas != 0 && r.ReadyReplicas == r.Replicas {
				app.ReadyWorkloadCount += 1
			}
		}

		resources = append(resources, r)
	}

	app.AppResources = resources
}

func (m *ApplicationManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	table, _, err := createOrGetAppTable(cluster.Name, namespace)
	if err != nil {
		log.Warnf("get application %s failed %s", ctx.Resource.GetID(), err.Error())
		return nil
	}

	app, err := getApplicationFromDB(table, ctx.Resource.GetID(), false)
	if err != nil {
		log.Warnf("get application %s failed %s", ctx.Resource.GetID(), err.Error())
		return nil
	}

	getAppResources(cluster.KubeClient, namespace, app)
	return genReturnApplication(app)
}

func getApplicationFromDB(table kvzoo.Table, appName string, system bool) (*types.Application, error) {
	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Commit()
	return getApplicationFromDBTx(tx, appName, system)
}

func getApplicationFromDBTx(tx kvzoo.Transaction, appName string, system bool) (*types.Application, error) {
	value, err := tx.Get(appName)
	if err != nil {
		return nil, err
	}

	var app types.Application
	if err := json.Unmarshal(value, &app); err != nil {
		return nil, err
	}

	if app.SystemChart != system {
		return nil, fmt.Errorf("user no authority to access application %s", appName)
	}

	return &app, nil
}

func (m *ApplicationManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	appName := ctx.Resource.GetID()
	table, _, err := createOrGetAppTable(cluster.Name, namespace)
	if err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("delete application %s failed: %s", appName, err.Error()))
	}

	if err := deleteApplication(table, cluster, namespace, appName, false); err != nil {
		if err == kvzoo.ErrNotFound {
			return resterror.NewAPIError(resterror.NotFound,
				fmt.Sprintf("application %s with namespace %s doesn't exist", appName, namespace))
		} else {
			return resterror.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("delete application %s failed: %s", appName, err.Error()))
		}
	}

	return nil
}

func deleteApplication(table kvzoo.Table, cluster *zke.Cluster, namespace, appName string, isSystemChart bool) error {
	app, err := updateAppStatusToDeleteFromDB(table, appName, isSystemChart)
	if err != nil {
		return err
	}

	go func() {
		if err := asyncDeleteApplication(table, cluster.KubeClient, namespace, app); err != nil {
			log.Warnf("delete application failed: %s", err.Error())
			publishApplicationEvent(cluster.Name, namespace, appName, deleteFailedReason, err.Error())
			updateAppStatusToFailed(table, cluster.Name, namespace, app)
		}
	}()

	return nil
}

func asyncDeleteApplication(table kvzoo.Table, cli client.Client, namespace string, app *types.Application) error {
	if err := deleteAppResources(cli, namespace, app.Manifests); err != nil {
		return fmt.Errorf("delete application %s resources failed: %s", app.Name, err.Error())
	}

	if err := deleteApplicationFromDB(table, app.GetID()); err != nil {
		return fmt.Errorf("delete application %s from db failed: %s", app.Name, err.Error())
	}

	return nil
}

func updateAppStatusToDeleteFromDB(table kvzoo.Table, name string, isSystemChart bool) (*types.Application, error) {
	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()
	app, err := getApplicationFromDBTx(tx, name, isSystemChart)
	if err != nil {
		return nil, err
	}

	if app.Status == types.AppStatusCreate || app.Status == types.AppStatusDelete {
		return nil, fmt.Errorf("application %s can`t delete when its status is %s", name, app.Status)
	}

	app.Status = types.AppStatusDelete
	app.SetDeletionTimestamp(time.Now())
	value, err := json.Marshal(app)
	if err != nil {
		return nil, err
	}

	if err := tx.Update(name, value); err != nil {
		return nil, err
	}

	return app, tx.Commit()
}

func deleteAppResources(cli client.Client, namespace string, manifests []types.Manifest) error {
	for _, manifest := range manifests {
		if manifest.Duplicate {
			continue
		}

		if err := helper.MapOnRuntimeObject(manifest.Content, func(ctx context.Context, obj runtime.Object) error {
			_, err := runtimeObjectToMetaObject(obj, namespace, true)
			if err != nil {
				return fmt.Errorf("runtime object to meta object with file %s failed: %s", manifest.File, err.Error())
			}

			if err := cli.Delete(ctx, obj, client.PropagationPolicy(metav1.DeletePropagationForeground)); err != nil {
				if apierrors.IsNotFound(err) == false {
					return fmt.Errorf("delete resource with file %s failed: %s", manifest.File, err.Error())
				}
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func deleteApplicationFromDB(table kvzoo.Table, name string) error {
	tx, err := table.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := tx.Delete(name); err != nil {
		return err
	}

	return tx.Commit()
}

func clearApplications(cli client.Client, clusterName, namespace string) error {
	table, tableName, err := createOrGetAppTable(clusterName, namespace)
	if err != nil {
		return err
	}

	apps, err := getApplicationsFromDB(table)
	if err != nil {
		return fmt.Errorf("get applications from db failed: %s", err.Error())
	}

	for _, app := range apps {
		app.Status = types.AppStatusDelete
		if err := updateApplicationToDB(table, app); err != nil {
			return fmt.Errorf("update application %s status to delete failed: %s", app.Name, err.Error())
		}

		if err := deleteAppResources(cli, namespace, app.Manifests); err != nil {
			return fmt.Errorf("delete application %s resources failed: %s", app.Name, err.Error())
		}
	}

	if err := db.GetGlobalDB().DeleteTable(tableName); err != nil {
		return fmt.Errorf("delete application table %s failed: %s", tableName, err.Error())
	}

	return nil
}

func genReturnApplication(app *types.Application) *types.Application {
	retApp := *app
	retApp.Configs = nil
	retApp.Manifests = nil
	return &retApp
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

func preInstallChartCRDs(cli client.Client, manifests []types.Manifest) error {
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
