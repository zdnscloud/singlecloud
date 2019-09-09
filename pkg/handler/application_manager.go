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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"helm.sh/helm/pkg/chart/loader"
	"helm.sh/helm/pkg/chartutil"
	"helm.sh/helm/pkg/engine"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
	"github.com/zdnscloud/gorest/api"
	restHandler "github.com/zdnscloud/gorest/api/handler"
	resttypes "github.com/zdnscloud/gorest/types"
	restutil "github.com/zdnscloud/gorest/util"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
	"github.com/zdnscloud/singlecloud/storage"
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
)

const (
	notesFileSuffix  = "NOTES.txt"
	ApplicationTable = "application"
)

type ApplicationManager struct {
	api.DefaultHandler
	clusters       *ClusterManager
	chartDir       string
	clusterEventCh <-chan interface{}
}

func newApplicationManager(clusters *ClusterManager, chartDir string) *ApplicationManager {
	m := &ApplicationManager{
		clusters:       clusters,
		chartDir:       chartDir,
		clusterEventCh: clusters.GetEventBus().Sub(eventbus.ClusterEvent),
	}
	go m.eventLoop()
	return m
}

func (m *ApplicationManager) eventLoop() {
	for {
		event := <-m.clusterEventCh
		switch e := event.(type) {
		case zke.DeleteCluster:
			if err := m.clusters.GetDB().DeleteTable(storage.GenTableName(ApplicationTable, e.Cluster.Name)); err != nil {
				log.Warnf("delete /application/cluster %s table failed: %s", e.Cluster.Name, err.Error())
			}
		}
	}
}

func (m *ApplicationManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	app := ctx.Object.(*types.Application)
	app.SetID(app.Name)
	if err := m.create(ctx, cluster, namespace, app); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource,
				fmt.Sprintf("duplicate chart %s with name %s", app.ChartName, app.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create application failed %s", err.Error()))
		}
	}

	retApp := *app
	retApp.Configs = nil
	retApp.Manifests = nil
	return &retApp, nil
}

func (m *ApplicationManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	isAdminUser := isAdmin(getCurrentUser(ctx))
	namespace := ctx.Object.GetParent().GetID()
	appValues, err := getApplicationsFromDB(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, namespace))
	if err != nil {
		log.Warnf("list applications failed %s", err.Error())
		return nil
	}

	var apps types.Applications
	for _, value := range appValues {
		if len(value) == 0 {
			continue
		}

		var app types.Application
		if err := json.Unmarshal(value, &app); err != nil {
			log.Warnf("list applications failed %s", err.Error())
			return nil
		}

		if isAdminUser == false && app.SystemChart {
			continue
		}

		app.Configs = nil
		app.Manifests = nil
		apps = append(apps, &app)
	}

	sort.Sort(apps)
	return apps
}

func (m *ApplicationManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	appName := ctx.Object.GetID()
	tableName := storage.GenTableName(ApplicationTable, cluster.Name, namespace)
	app, err := updateApplicationStatusFromDB(m.clusters.GetDB(), getCurrentUser(ctx), tableName, appName, types.AppStatusDelete)
	if err != nil {
		if err == storage.ErrNotFoundResource {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("application %s with namespace %s doesn't exist", appName, namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("delete application %s failed: %s", appName, err.Error()))
		}
	}

	go deleteApplication(m.clusters.GetDB(), cluster.KubeClient, tableName, namespace, app)
	return nil
}

func deleteApplication(db storage.DB, cli client.Client, tableName, namespace string, app *types.Application) {
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
}

func updateApplicationStatusFromDB(db storage.DB, userName, tableName, name, status string) (*types.Application, error) {
	tx, err := BeginTableTransaction(db, tableName)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()
	value, err := tx.Get(name)
	if err != nil {
		return nil, err
	}

	var app types.Application
	if err := json.Unmarshal(value, &app); err != nil {
		return nil, err
	}

	if isAdmin(userName) == false && app.SystemChart {
		return nil, fmt.Errorf("user %s no authority to delete application %s", userName, name)
	}

	if status == types.AppStatusDelete && (app.Status == types.AppStatusCreate || app.Status == types.AppStatusDelete) {
		return nil, fmt.Errorf("application %s can`t delete when its status is %s", name, app.Status)
	}

	app.Status = status
	value, err = json.Marshal(app)
	if err != nil {
		return nil, err
	}

	if err := tx.Update(name, value); err != nil {
		return nil, err
	}

	return &app, tx.Commit()
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

func deleteApplicationFromDB(db storage.DB, tableName, name string) error {
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

func (m *ApplicationManager) create(ctx *resttypes.Context, cluster *zke.Cluster, namespace string, app *types.Application) error {
	if exists, err := hasNamespace(cluster.KubeClient, namespace); err != nil {
		return fmt.Errorf("check namespace %s if exists failed: %s", namespace, err.Error())
	} else if exists == false {
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

	manifests, err := loadChartFiles(ctx, namespace, chartPath, app, DefaultCapabilities)
	if err != nil {
		return fmt.Errorf("load chart %s with version %s files failed: %s", app.ChartName, app.ChartVersion, err.Error())
	}

	app.Manifests = manifests
	app.Status = types.AppStatusCreate
	app.SetCreationTimestamp(time.Now())
	app.ChartIcon = genChartIcon(app.ChartName)
	tableName := storage.GenTableName(ApplicationTable, cluster.Name, namespace)
	if err := addOrUpdateAppToDB(m.clusters.GetDB(), tableName, app, true); err != nil {
		return fmt.Errorf("add application %s to db failed: %s", app.Name, err.Error())
	}

	go createApplication(m.clusters.GetDB(), cluster.KubeClient, isAdminUser, tableName, namespace,
		genUrlPrefix(ctx, cluster.Name), app)
	return nil
}

func addOrUpdateAppToDB(db storage.DB, tableName string, app *types.Application, isCreate bool) error {
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

func getApplicationsFromDB(db storage.DB, tableName string) (map[string][]byte, error) {
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

	return appValues, nil
}

func getApplicationsFromDBByChartName(db storage.DB, tableName, chartName string) ([]*types.Application, error) {
	apps := []*types.Application{}

	appValues, err := getApplicationsFromDB(db, tableName)
	if err != nil {
		return nil, err
	}

	for _, appValue := range appValues {
		app := &types.Application{}
		if err := json.Unmarshal(appValue, app); err != nil {
			continue
		}
		if app.ChartName == chartName {
			apps = append(apps, app)
		}
	}

	return apps, nil
}

func getApplicationFromDB(db storage.DB, tableName, appName string) (*types.Application, error) {
	tx, err := BeginTableTransaction(db, tableName)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()
	appValue, err := tx.Get(appName)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	app := types.Application{}
	if err := json.Unmarshal(appValue, &app); err != nil {
		return nil, fmt.Errorf("Unmarshal application %s failed %s", appName, err.Error())
	}

	return &app, nil
}

func loadChartFiles(ctx *resttypes.Context, namespace, chartPath string, app *types.Application, caps *chartutil.Capabilities) ([]types.Manifest, error) {
	rawValues, err := getChartValues(ctx, app)
	if err != nil {
		return nil, err
	}
	if rawValues == nil {
		rawValues = make(map[string]interface{})
	}

	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}

	options := chartutil.ReleaseOptions{
		Name:      app.Name,
		Namespace: namespace,
		IsInstall: true,
	}
	valuesToRender, err := chartutil.ToRenderValues(chartRequested, rawValues, options, caps)
	if err != nil {
		return nil, err
	}
	if rel, ok := valuesToRender["Release"].(map[string]interface{}); ok {
		rel["Service"] = "singlecloud"
	}

	files, err := engine.Render(chartRequested, valuesToRender)
	if err != nil {
		return nil, err
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

	return manifests, nil
}

func createApplication(db storage.DB, cli client.Client, isAdmin bool, tableName, namespace, urlPrefix string, app *types.Application) {
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

			switch typ := strings.ToLower(gvk.Kind); typ {
			case types.DeploymentType, types.DaemonSetType, types.StatefulSetType,
				types.ConfigMapType, types.SecretType, types.ServiceType, types.IngressType,
				types.CronJobType, types.JobType:
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

func getChartValues(ctx *resttypes.Context, app *types.Application) (map[string]interface{}, error) {
	if app.Configs == nil {
		return nil, nil
	}

	schema := ctx.Schemas.Schema(&ctx.Object.GetSchema().Version, app.ChartName)
	if schema == nil {
		return nil, fmt.Errorf("no found schema %s", app.ChartName)
	}

	chartCtx := &resttypes.Context{
		Schemas: ctx.Schemas,
		Object: &resttypes.Resource{
			Schema: schema,
		},
	}

	val := schema.StructVal
	valPtr := reflect.New(val.Type())
	valPtr.Elem().Set(val)
	obj := valPtr.Interface()
	if err := json.Unmarshal(app.Configs, obj); err != nil {
		return nil, fmt.Errorf("unmarshal application %s config failed: %v", app.Name, err.Error())
	}

	m, err := restHandler.ObjectToMap(chartCtx, obj)
	if err != nil {
		return nil, fmt.Errorf("create application %s config is invalid: %v", app.ChartName, err.Error())
	}
	return m, nil
}

func genUrlPrefix(ctx *resttypes.Context, clusterName string) string {
	req := ctx.Request
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	urls := strings.SplitAfterN(req.URL.Path, fmt.Sprintf("/clusters/%s/namespaces/", clusterName), 2)
	if len(urls) == 2 {
		return fmt.Sprintf("%s://%s%s", scheme, req.Host, urls[0])
	} else {
		apiVersion := ctx.Object.GetSchema().Version
		return path.Join(fmt.Sprintf("%s://%s", scheme, req.Host),
			resttypes.GroupPrefix, apiVersion.Group, apiVersion.Version,
			fmt.Sprintf("/clusters/%s/namespaces/", clusterName))
	}
}

func clearApplications(db storage.DB, cli client.Client, clusterName, namespace string) error {
	tableName := storage.GenTableName(ApplicationTable, clusterName, namespace)
	appValues, err := getApplicationsFromDB(db, tableName)
	if err != nil {
		return fmt.Errorf("get applications from db failed: %s", err.Error())
	}

	for name, value := range appValues {
		if len(value) == 0 {
			continue
		}

		var app types.Application
		if err := json.Unmarshal(value, &app); err != nil {
			return fmt.Errorf("unmarshal application %s failed: %s", name, err.Error())
		}

		app.Status = types.AppStatusDelete
		if err := addOrUpdateAppToDB(db, tableName, &app, false); err != nil {
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
