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
	"github.com/zdnscloud/singlecloud/pkg/types"
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
	notesFileSuffix = "NOTES.txt"
	AppTableSuffix  = "application"
)

type ApplicationManager struct {
	api.DefaultHandler
	clusters *ClusterManager
	chartDir string
}

func newApplicationManager(clusters *ClusterManager, chartDir string) *ApplicationManager {
	return &ApplicationManager{clusters: clusters, chartDir: chartDir}
}

func (m *ApplicationManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	app := ctx.Object.(*types.Application)
	app.SetID(app.Name)
	urlPrefix := genUrlPrefix(ctx, cluster.Name, namespace)
	if err := m.create(ctx, cluster.KubeClient, namespace, urlPrefix, app); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource,
				fmt.Sprintf("duplicate chart %s with name %s", app.ChartName, app.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create application failed %s", err.Error()))
		}
	}

	retApp := *app
	retApp.Configs = ""
	retApp.Manifests = nil
	return &retApp, nil
}

func (m *ApplicationManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	tx, err := BeginTableTransaction(m.clusters.GetDB(), namespace+"_"+AppTableSuffix)
	if err != nil {
		log.Warnf("list applications failed %s", err.Error())
		return nil
	}

	appValues, err := tx.List()
	if err != nil {
		tx.Rollback()
		log.Warnf("list applications failed %s", err.Error())
		return nil
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
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

		app.Configs = ""
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
	app, err := updateApplicationStatusFromDB(m.clusters.GetDB(), namespace, appName, types.AppStatusDelete)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("application %s with namespace %s doesn't exist", appName, namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("delete application %s failed: %s", appName, err.Error()))
		}
	}

	go deleteApplication(m.clusters.GetDB(), cluster.KubeClient, namespace, app)
	return nil
}

func deleteApplication(db storage.DB, cli client.Client, namespace string, app *types.Application) {
	for fileName, content := range app.Manifests {
		if err := helper.MapOnRuntimeObject(content, func(ctx context.Context, obj runtime.Object) error {
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				return fmt.Errorf("runtime object to meta object with file %s failed: %s", fileName, err.Error())
			}

			metaObj.SetNamespace(namespace)
			if err := cli.Delete(ctx, obj, client.PropagationPolicy(metav1.DeletePropagationForeground)); err != nil {
				if apierrors.IsNotFound(err) == false {
					return fmt.Errorf("delete resource with file %s failed: %s", fileName, err.Error())
				}
			}

			return nil
		}); err != nil {
			app.Status = types.AppStatusFailed
			if err := addOrUpdateAppToDB(db, namespace, app, false); err != nil {
				log.Warnf("delete application %s resources failed, update status get error: %s", app.Name, err.Error())
			}
			log.Warnf("delete application %s resource with file %s failed: %s", app.Name, fileName, err.Error())
			return
		}
	}

	if err := deleteApplicationFromDB(db, namespace, app.GetID()); err != nil {
		app.Status = types.AppStatusFailed
		if err := addOrUpdateAppToDB(db, namespace, app, false); err != nil {
			log.Warnf("delete application %s failed, update status get error: %s", app.Name, err.Error())
		}
		log.Warnf("delete application %s from db failed: %s", app.Name, err.Error())
	}
}

func updateApplicationStatusFromDB(db storage.DB, namespace, name, status string) (*types.Application, error) {
	tx, err := BeginTableTransaction(db, namespace+"_"+AppTableSuffix)
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

func deleteApplicationFromDB(db storage.DB, namespace, name string) error {
	tx, err := BeginTableTransaction(db, namespace+"_"+AppTableSuffix)
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := tx.Delete(name); err != nil {
		return err
	}

	return tx.Commit()
}

func (m *ApplicationManager) create(ctx *resttypes.Context, cli client.Client, namespace, url string, app *types.Application) error {
	chartPath := path.Join(m.chartDir, app.ChartName, app.ChartVersion)
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		return err
	}

	files, err := loadChartFiles(ctx, namespace, chartPath, app)
	if err != nil {
		return fmt.Errorf("load chart %s with version %s files failed: %s", app.ChartName, app.ChartVersion, err.Error())
	}

	app.Manifests = files
	app.Status = types.AppStatusCreate
	app.SetCreationTimestamp(time.Now())
	if err := addOrUpdateAppToDB(m.clusters.GetDB(), namespace, app, true); err != nil {
		return fmt.Errorf("add application %s to db failed: %s", app.Name, err.Error())
	}

	go createApplication(m.clusters.GetDB(), cli, namespace, url, app)
	return nil
}

func addOrUpdateAppToDB(db storage.DB, namespace string, app *types.Application, isCreate bool) error {
	value, err := json.Marshal(app)
	if err != nil {
		return fmt.Errorf("marshal application %s failed: %s", app.Name, err.Error())
	}

	tx, err := BeginTableTransaction(db, namespace+"_"+AppTableSuffix)
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

func loadChartFiles(ctx *resttypes.Context, namespace, chartPath string, app *types.Application) (map[string]string, error) {
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
	valuesToRender, err := chartutil.ToRenderValues(chartRequested, rawValues, options, DefaultCapabilities)
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

	for fileName, _ := range files {
		if strings.HasSuffix(fileName, notesFileSuffix) {
			delete(files, fileName)
		}
	}

	return files, nil
}

func createApplication(db storage.DB, cli client.Client, namespace, url string, app *types.Application) {
	appResources, err := createAppResources(cli, namespace, url, app.Manifests)
	if err != nil {
		app.Status = types.AppStatusFailed
		if err := addOrUpdateAppToDB(db, namespace, app, false); err != nil {
			log.Warnf("create application %s resoures failed, update status get error: %s", app.Name, err.Error())
		}
		log.Warnf("create application %s resources failed: %s", app.Name, err.Error())
		return
	}

	app.AppResources = appResources
	app.Status = types.AppStatusSucceed
	if err := addOrUpdateAppToDB(db, namespace, app, false); err != nil {
		app.Status = types.AppStatusFailed
		if err := addOrUpdateAppToDB(db, namespace, app, false); err != nil {
			log.Warnf("update application %s status failed, update status get error: %s", app.Name, err.Error())
		}
		log.Warnf("update application %s status to succeed failed: %s", app.Name, err.Error())
	}
}

func createAppResources(cli client.Client, namespace, urlPrefix string, manifests map[string]string) (types.AppResources, error) {
	var appResources types.AppResources
	for fileName, content := range manifests {
		if err := helper.MapOnRuntimeObject(content, func(ctx context.Context, obj runtime.Object) error {
			gvk := obj.GetObjectKind().GroupVersionKind()
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				return fmt.Errorf("runtime object to meta object with file %s failed: %s", fileName, err.Error())
			}

			metaObj.SetNamespace(namespace)
			if err := cli.Create(ctx, obj); err != nil {
				return fmt.Errorf("create resource with file %s failed: %s", fileName, err.Error())
			}

			switch typ := strings.ToLower(gvk.Kind); typ {
			case types.DeploymentType, types.DaemonSetType, types.StatefulSetType,
				types.ConfigMapType, types.SecretType, types.ServiceType, types.IngressType,
				types.CronJobType, types.JobType:
				appResources = append(appResources, types.AppResource{
					Name: metaObj.GetName(),
					Type: typ,
					Link: path.Join(urlPrefix, restutil.GuessPluralName(typ)),
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
	if app.Configs == "" {
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
	if err := json.Unmarshal([]byte(app.Configs), obj); err != nil {
		return nil, fmt.Errorf("unmarshal application %s config failed: %v", app.Name, err.Error())
	}

	m, err := restHandler.ObjectToMap(chartCtx, obj)
	if err != nil {
		return nil, fmt.Errorf("create application %s config is invalid: %v", app.ChartName, err.Error())
	}

	return m, nil
}

func genUrlPrefix(ctx *resttypes.Context, clusterName, namespace string) string {
	req := ctx.Request
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	urls := strings.SplitAfterN(req.URL.Path, "/namespaces/"+namespace, 2)
	if len(urls) == 2 {
		return fmt.Sprintf("%s://%s%s", scheme, req.Host, urls[0])
	} else {
		apiVersion := ctx.Object.GetSchema().Version
		return path.Join(fmt.Sprintf("%s://%s", scheme, req.Host),
			resttypes.GroupPrefix, apiVersion.Group, apiVersion.Version,
			fmt.Sprintf("/clusters/%s/namespaces/%s", clusterName, namespace))
	}
}
