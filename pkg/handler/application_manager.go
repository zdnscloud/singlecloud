package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"time"

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
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

var (
	DefaultCapabilities = &chartutil.Capabilities{
		KubeVersion: chartutil.KubeVersion{
			Version: "v1.13.0",
			Major:   "1",
			Minor:   "13",
		},
		APIVersions: chartutil.DefaultVersionSet,
	}

	AppSupportResourceTypes = []string{"deployment", "daemonset", "statefulset", "cronjob", "job", "configmap", "secret", "service", "ing    ress"}
)

const (
	notesFileSuffix  = "NOTES.txt"
	ApplicationTable = "application"

	crdCheckTimes    = 20
	crdCheckInterval = 5 * time.Second
)

type ApplicationManager struct {
	clusters       *ClusterManager
	chartDir       string
	clusterEventCh <-chan interface{}
	chartConfigs   map[string]chartConfig
}

type chartConfig struct {
	structVal reflect.Value
	fields    resourcefield.ResourceField
}

func newApplicationManager(clusters *ClusterManager, chartDir string) *ApplicationManager {
	m := &ApplicationManager{
		clusters:       clusters,
		chartDir:       chartDir,
		clusterEventCh: clusters.GetEventBus().Sub(eventbus.ClusterEvent),
		chartConfigs:   make(map[string]chartConfig),
	}
	go m.eventLoop()
	return m
}

func (m *ApplicationManager) eventLoop() {
	for {
		event := <-m.clusterEventCh
		switch e := event.(type) {
		case zke.DeleteCluster:
			tn, _ := kvzoo.TableNameFromSegments(ApplicationTable, e.Cluster.Name)
			if err := m.clusters.GetDB().DeleteTable(tn); err != nil {
				log.Warnf("delete /application/cluster %s table failed: %s", e.Cluster.Name, err.Error())
			}
		}
	}
}

func (m *ApplicationManager) addChartsConfig(chartsConfig []interface{}) error {
	for _, config := range chartsConfig {
		typ := reflect.TypeOf(config)
		fields, err := resourcefield.New(typ)
		if err != nil {
			return err
		}

		m.chartConfigs[strings.ToLower(typ.Name())] = chartConfig{
			structVal: reflect.ValueOf(config),
			fields:    fields,
		}
	}
	return nil
}

func (m *ApplicationManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	app := ctx.Resource.(*types.Application)
	app.SetID(app.Name)
	if err := m.create(ctx, cluster, namespace, app); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource,
				fmt.Sprintf("duplicate chart %s with name %s", app.ChartName, app.Name))
		} else {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create application failed %s", err.Error()))
		}
	}

	return genReturnApplication(app), nil
}

func (m *ApplicationManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	tn, _ := kvzoo.TableNameFromSegments(ApplicationTable, cluster.Name, namespace)
	allApps, err := getApplicationsFromDB(m.clusters.GetDB(), tn)
	if err != nil {
		log.Warnf("list applications failed %s", err.Error())
		return nil
	}

	var apps types.Applications
	for _, app := range allApps {
		if app.SystemChart {
			continue
		}

		apps = append(apps, genReturnApplication(app))
	}

	sort.Sort(apps)
	return apps
}

func (m *ApplicationManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Resource.GetParent().GetID()
	tableName, _ := kvzoo.TableNameFromSegments(ApplicationTable, cluster.Name, namespace)
	app, err := getApplicationFromDB(m.clusters.GetDB(), tableName, ctx.Resource.GetID(), false)
	if err != nil {
		log.Warnf("get applications failed %s", err.Error())
		return nil
	}

	return genReturnApplication(app)
}

func (m *ApplicationManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	appName := ctx.Resource.GetID()
	if err := deleteApplication(m.clusters.GetDB(), cluster.KubeClient, cluster.Name, namespace, appName, false); err != nil {
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

func deleteApplication(db kvzoo.DB, cli client.Client, clusterName, namespace, appName string, isSystemChart bool) error {
	tableName, _ := kvzoo.TableNameFromSegments(ApplicationTable, clusterName, namespace)
	app, err := updateAppStatusToDeleteFromDB(db, tableName, appName, isSystemChart)
	if err != nil {
		return err
	}

	go func() {
		if err := deleteAppResources(cli, namespace, app.Manifests); err != nil {
			app.Status = types.AppStatusFailed
			if err := addOrUpdateAppToDB(db, tableName, app, false); err != nil {
				log.Warnf("delete application %s resources failed, update status get error: %s", app.Name, err.Error())
			}
			log.Warnf("delete application %s resources failed: %s", app.Name, err.Error())
			return
		}

		if err := deleteApplicationFromDB(db, tableName, app.GetID()); err != nil {
			app.Status = types.AppStatusFailed
			if err := addOrUpdateAppToDB(db, tableName, app, false); err != nil {
				log.Warnf("delete application %s failed, update status get error: %s", app.Name, err.Error())
			}

			log.Warnf("delete application %s from db failed: %s", app.Name, err.Error())
		}
	}()

	return nil
}

func updateAppStatusToDeleteFromDB(db kvzoo.DB, tableName kvzoo.TableName, name string, isSystemChart bool) (*types.Application, error) {
	tx, err := BeginTableTransaction(db, tableName)
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
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				return fmt.Errorf("runtime object to meta object with file %s failed: %s", manifest.File, err.Error())
			}

			if metaObj.GetNamespace() == "" {
				metaObj.SetNamespace(namespace)
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

func deleteApplicationFromDB(db kvzoo.DB, tableName kvzoo.TableName, name string) error {
	tx, err := BeginTableTransaction(db, tableName)
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := tx.Delete(name); err != nil {
		return err
	}

	return tx.Commit()
}

func (m *ApplicationManager) create(ctx *resource.Context, cluster *zke.Cluster, namespace string, app *types.Application) error {
	if hasNamespace(cluster.KubeClient, namespace) == false {
		return fmt.Errorf("namespace %s is not found", namespace)
	}

	chartPath := path.Join(m.chartDir, app.ChartName, app.ChartVersion)
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		return err
	}

	info, err := getChartInfo(chartPath)
	if err != nil {
		return fmt.Errorf("load chart %s with version %s info failed: %s", app.ChartName, app.ChartVersion, err.Error())
	}

	isAdminUser := isAdmin(getCurrentUser(ctx))
	if isAdminUser == false && info.SystemChart {
		return fmt.Errorf("user %s no authority to create application with chart %s", getCurrentUser(ctx), app.ChartName)
	}

	app.SystemChart = info.SystemChart

	if clusterVersion, err := cluster.KubeClient.ServerVersion(); err != nil {
		return fmt.Errorf("get cluster %s version failed: %s", cluster.Name, err.Error())
	} else {
		DefaultCapabilities.KubeVersion.Version = clusterVersion.GitVersion
		DefaultCapabilities.KubeVersion.Major = clusterVersion.Major
		DefaultCapabilities.KubeVersion.Minor = clusterVersion.Minor
	}

	configs, err := m.parseChartConfigs(app.ChartName, app.Configs)
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
	app.ChartIcon = genChartIcon(app.ChartName)
	tableName, _ := kvzoo.TableNameFromSegments(ApplicationTable, cluster.Name, namespace)
	if err := addOrUpdateAppToDB(m.clusters.GetDB(), tableName, app, true); err != nil {
		return fmt.Errorf("add application %s to db failed: %s", app.Name, err.Error())
	}

	go createApplication(m.clusters.GetDB(), cluster.KubeClient, isAdminUser, tableName, namespace,
		genUrlPrefix(ctx, cluster.Name), app, crdManifests)
	return nil
}

func addOrUpdateAppToDB(db kvzoo.DB, tableName kvzoo.TableName, app *types.Application, isCreate bool) error {
	value, err := json.Marshal(app)
	if err != nil {
		return fmt.Errorf("marshal application %s failed: %s", app.Name, err.Error())
	}

	tx, err := BeginTableTransaction(db, tableName)
	if err != nil {
		return err
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

func getApplicationsFromDB(db kvzoo.DB, tableName kvzoo.TableName) (types.Applications, error) {
	tx, err := BeginTableTransaction(db, tableName)
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

func getApplicationFromDB(db kvzoo.DB, tableName kvzoo.TableName, appName string, isSystemChart bool) (*types.Application, error) {
	tx, err := BeginTableTransaction(db, tableName)
	if err != nil {
		return nil, err
	}

	defer tx.Commit()
	return getApplicationFromDBTx(tx, appName, isSystemChart)
}

func getApplicationFromDBTx(tx kvzoo.Transaction, appName string, isSystemChart bool) (*types.Application, error) {
	value, err := tx.Get(appName)
	if err != nil {
		return nil, err
	}

	var app types.Application
	if err := json.Unmarshal(value, &app); err != nil {
		return nil, err
	}

	//all user can`t access system chart, system chart operate is another logic and in alone page
	if app.SystemChart != isSystemChart {
		return nil, fmt.Errorf("user no authority to access application %s", appName)
	}

	return &app, nil
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

func createApplication(db kvzoo.DB, cli client.Client, isAdmin bool, tableName kvzoo.TableName, namespace, urlPrefix string, app *types.Application, crdManifests []types.Manifest) {
	if err := preInstallChartCRDs(cli, crdManifests); err != nil {
		app.Status = types.AppStatusFailed
		if err := addOrUpdateAppToDB(db, tableName, app, false); err != nil {
			log.Warnf("create application %s crds failed, update status get error: %s", app.Name, err.Error())
		}
		log.Warnf("create application %s crds failed: %s", app.Name, err.Error())
		return
	}

	appResources, err := createAppResources(cli, isAdmin, namespace, urlPrefix, app.Manifests)
	if err != nil {
		app.Status = types.AppStatusFailed
		if err := addOrUpdateAppToDB(db, tableName, app, false); err != nil {
			log.Warnf("create application %s resoures failed, update status get error: %s", app.Name, err.Error())
		}
		log.Warnf("create application %s resources failed: %s", app.Name, err.Error())
		return
	}

	app.AppResources = appResources
	app.Status = types.AppStatusSucceed
	if err := addOrUpdateAppToDB(db, tableName, app, false); err != nil {
		app.Status = types.AppStatusFailed
		if err := addOrUpdateAppToDB(db, tableName, app, false); err != nil {
			log.Warnf("update application %s status failed, update status get error: %s", app.Name, err.Error())
		}
		log.Warnf("update application %s status to succeed failed: %s", app.Name, err.Error())
	}
}

func createAppResources(cli client.Client, isAdmin bool, namespace, urlPrefix string, manifests []types.Manifest) (types.AppResources, error) {
	var appResources types.AppResources
	for i, manifest := range manifests {
		if err := helper.MapOnRuntimeObject(manifest.Content, func(ctx context.Context, obj runtime.Object) error {
			if obj == nil {
				return fmt.Errorf("cann`t unmarshal file %s to k8s runtime object\n", manifest.File)
			}

			gvk := obj.GetObjectKind().GroupVersionKind()
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				return fmt.Errorf("runtime object to meta object with file %s failed: %s", manifest.File, err.Error())
			}

			tmpNS := namespace
			if nm := metaObj.GetNamespace(); nm != "" {
				if isAdmin == false {
					return fmt.Errorf("chart file %s should not has namespace", manifest.File)
				}
				tmpNS = nm
			} else {
				metaObj.SetNamespace(namespace)
			}

			if err := cli.Create(ctx, obj); err != nil {
				if apierrors.IsAlreadyExists(err) {
					manifests[i].Duplicate = true
				}
				return fmt.Errorf("create resource with file %s failed: %s", manifest.File, err.Error())
			}

			typ := strings.ToLower(gvk.Kind)
			if slice.SliceIndex(AppSupportResourceTypes, typ) != -1 {
				appResources = append(appResources, types.AppResource{
					Name: metaObj.GetName(),
					Type: typ,
					Link: path.Join(urlPrefix, tmpNS, restutil.GuessPluralName(typ), metaObj.GetName()),
				})
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	sort.Sort(appResources)
	return appResources, nil
}

func (m *ApplicationManager) parseChartConfigs(chartName string, appConfigs json.RawMessage) (map[string]interface{}, error) {
	objMap := make(map[string]interface{})
	if appConfigs == nil {
		return objMap, nil
	}

	if err := json.Unmarshal(appConfigs, &objMap); err != nil {
		return nil, fmt.Errorf("unmarshal chart %s configs failed: %v", chartName, err.Error())
	}

	chartConfig, ok := m.chartConfigs[strings.ToLower(chartName)]
	if ok == false {
		return nil, fmt.Errorf("no found chart %s resource info", chartName)
	}

	if chartConfig.fields != nil {
		if err := chartConfig.fields.CheckRequired(objMap); err != nil {
			return nil, err
		}
	}

	val := chartConfig.structVal
	valPtr := reflect.New(val.Type())
	valPtr.Elem().Set(val)
	obj := valPtr.Interface()
	if err := json.Unmarshal(appConfigs, obj); err != nil {
		return nil, fmt.Errorf("unmarshal chart %s configs failed: %v", chartName, err.Error())
	}

	if chartConfig.fields != nil {
		if err := chartConfig.fields.Validate(obj); err != nil {
			return nil, err
		}
	}

	return objMap, nil
}

func genUrlPrefix(ctx *resource.Context, clusterName string) string {
	req := ctx.Request
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	urls := strings.SplitAfterN(req.URL.Path, fmt.Sprintf("/clusters/%s/namespaces/", clusterName), 2)
	if len(urls) == 2 {
		return fmt.Sprintf("%s://%s%s", scheme, req.Host, urls[0])
	} else {
		return path.Join(fmt.Sprintf("%s://%s", scheme, req.Host),
			resource.GroupPrefix, Version.Group, Version.Version,
			fmt.Sprintf("/clusters/%s/namespaces/", clusterName))
	}
}

func clearApplications(db kvzoo.DB, cli client.Client, clusterName, namespace string) error {
	tableName, _ := kvzoo.TableNameFromSegments(ApplicationTable, clusterName, namespace)
	apps, err := getApplicationsFromDB(db, tableName)
	if err != nil {
		return fmt.Errorf("get applications from db failed: %s", err.Error())
	}

	for _, app := range apps {
		app.Status = types.AppStatusDelete
		if err := addOrUpdateAppToDB(db, tableName, app, false); err != nil {
			return fmt.Errorf("update application %s status to delete failed: %s", app.Name, err.Error())
		}

		if err := deleteAppResources(cli, namespace, app.Manifests); err != nil {
			return fmt.Errorf("delete application %s resources failed: %s", app.Name, err.Error())
		}
	}

	if err := db.DeleteTable(tableName); err != nil {
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
				// This indicates a naming conflict, but it's probably not the
				// job of this function to fail because of that. Instead,
				// we treat it as a success, since the process should be able to
				// continue.
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
