package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"helm.sh/helm/pkg/chart/loader"
	"helm.sh/helm/pkg/chartutil"
	"helm.sh/helm/pkg/engine"
	"helm.sh/helm/pkg/release"
	"helm.sh/helm/pkg/releaseutil"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
	"github.com/zdnscloud/gorest/api"
	restHandler "github.com/zdnscloud/gorest/api/handler"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var (
	DefaultCapabilities = &chartutil.Capabilities{
		KubeVersion: chartutil.KubeVersion{
			Version: "v1.13.0",
			Major:   "1",
			Minor:   "13",
		},
	}
	magicGzip = []byte{0x1f, 0x8b, 0x08}
)

const (
	notesFileSuffix = "NOTES.txt"
	cmReleaseKey    = "application-release"
	cmLabelChart    = "chart-name"
	cmLabelVersion  = "chart-version"
	cmLabelOwnerApp = "application-name"
	cmLabelRevision = "application-revision"
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
	if err := createApplication(ctx, cluster.KubeClient, namespace, m.chartDir, app); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource,
				fmt.Sprintf("duplicate chart %s with name %s", app.ChartName, app.Name))
		} else {
			return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("install chart failed %s", err.Error()))
		}
	}

	app.Configs = ""
	app.SetID(app.Name)
	return app, nil
}

func (m *ApplicationManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sConfigMaps, err := getConfigMaps(cluster.KubeClient, namespace)
	if err != nil {
		log.Warnf("list applications info failed:%s", err.Error())
		return nil
	}

	if len(k8sConfigMaps.Items) == 0 {
		log.Debugf("no install any applications")
		return nil
	}

	var apps types.Applications
	for _, item := range k8sConfigMaps.Items {
		app, err := k8sConfigMapToScApplication(cluster.KubeClient, &item)
		if err != nil {
			log.Warnf("list applications info failed:%s", err.Error())
			continue
		}

		if app != nil {
			apps = append(apps, app)
		}
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
	if err := deleteApplication(cluster.KubeClient, namespace, appName); err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("application %s with namespace %s doesn't exist", appName, namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("uninstall application %s failed: %s", appName, err.Error()))
		}
	}

	return nil
}

func k8sConfigMapToScApplication(cli client.Client, k8sConfigMap *corev1.ConfigMap) (*types.Application, error) {
	release, err := decodeRelease(k8sConfigMap.Data[cmReleaseKey])
	if err != nil {
		return nil, err
	}

	if release == nil {
		return nil, nil
	}

	chartVersion := k8sConfigMap.Labels[cmLabelVersion]
	if release.Chart.Metadata != nil {
		chartVersion = release.Chart.Metadata.Version
	}

	var appResources types.AppResources
	manifests := releaseutil.SplitManifests(release.Manifest)
	for _, content := range manifests {
		var contentMap map[string]interface{}
		if err := yaml.Unmarshal([]byte(content), &contentMap); err != nil {
			return nil, fmt.Errorf("unmarshal application resource content failed: %s", err.Error())
		}

		if len(contentMap) == 0 {
			continue
		}

		metadata, ok := contentMap["metadata"].(map[interface{}]interface{})
		if ok == false {
			return nil, fmt.Errorf("invalid resource without metadata")
		}
		resourceName := metadata["name"].(string)

		switch strings.ToLower(contentMap["kind"].(string)) {
		case types.DeploymentType:
			k8sDeploy, err := getDeployment(cli, k8sConfigMap.Namespace, resourceName)
			if err != nil {
				return nil, fmt.Errorf("get deploy %s failed: %v", resourceName, err.Error())
			}

			deploy, err := k8sDeployToSCDeploy(cli, k8sDeploy)
			if err != nil {
				return nil, fmt.Errorf("trans k8sdeploy to scdeploy failed: %v", err.Error())
			}
			appResources.Deployments = append(appResources.Deployments, *deploy)
		case types.StatefulSetType:
			k8sStatefulSet, err := getStatefulSet(cli, k8sConfigMap.Namespace, resourceName)
			if err != nil {
				return nil, fmt.Errorf("get statefulset %s failed: %v", resourceName, err.Error())
			}
			appResources.StatefulSets = append(appResources.StatefulSets, *k8sStatefulSetToSCStatefulSet(k8sStatefulSet))
		case types.DaemonSetType:
			k8sDaemonSet, err := getDaemonSet(cli, k8sConfigMap.Namespace, resourceName)
			if err != nil {
				return nil, fmt.Errorf("get daemonset %s failed: %v", resourceName, err.Error())
			}

			daemonset, err := k8sDaemonSetToSCDaemonSet(cli, k8sDaemonSet)
			if err != nil {
				return nil, fmt.Errorf("trans k8sdaemonset to scdaemonset failed: %v", err.Error())
			}
			appResources.DaemonSets = append(appResources.DaemonSets, *daemonset)
		case types.ConfigMapType:
			configMap, err := getConfigMap(cli, k8sConfigMap.Namespace, resourceName)
			if err != nil {
				return nil, fmt.Errorf("get configmap %s failed: %v", resourceName, err.Error())
			}

			appResources.ConfigMaps = append(appResources.ConfigMaps, *k8sConfigMapToSCConfigMap(configMap))
		case types.SecretType:
			k8sSecret, err := getSecret(cli, k8sConfigMap.Namespace, resourceName)
			if err != nil {
				return nil, fmt.Errorf("get secret %s failed: %v", resourceName, err.Error())
			}
			appResources.Secrets = append(appResources.Secrets, *k8sSecretToSCSecret(k8sSecret))
		case types.ServiceType:
			k8sService, err := getService(cli, k8sConfigMap.Namespace, resourceName)
			if err != nil {
				return nil, fmt.Errorf("get service %s failed: %v", resourceName, err.Error())
			}
			appResources.Services = append(appResources.Services, *k8sServiceToSCService(k8sService))
		case types.IngressType:
			k8sIngress, err := getIngress(cli, k8sConfigMap.Namespace, resourceName)
			if err != nil {
				return nil, fmt.Errorf("get ingress %s failed: %v", resourceName, err.Error())
			}
			appResources.Ingresses = append(appResources.Ingresses, *k8sIngressToSCIngress(k8sIngress))
		case types.CronJobType:
			k8sCronJob, err := getCronJob(cli, k8sConfigMap.Namespace, resourceName)
			if err != nil {
				return nil, fmt.Errorf("get cronjob %s failed: %v", resourceName, err.Error())
			}
			appResources.CronJobs = append(appResources.CronJobs, *k8sCronJobToScCronJob(k8sCronJob))
		case types.JobType:
			k8sJob, err := getJob(cli, k8sConfigMap.Namespace, resourceName)
			if err != nil {
				return nil, fmt.Errorf("get job %s failed: %v", resourceName, err.Error())
			}
			appResources.Jobs = append(appResources.Jobs, *k8sJobToSCJob(k8sJob))
		}
	}

	app := &types.Application{
		Name:         release.Name,
		Version:      release.Version,
		ChartName:    release.Chart.Name(),
		ChartVersion: chartVersion,
		AppResources: appResources,
	}
	app.SetID(app.Name)
	app.SetType(types.ApplicationType)
	app.SetCreationTimestamp(k8sConfigMap.CreationTimestamp.Time)
	return app, nil
}

func deleteApplication(cli client.Client, namespace, name string) error {
	k8sConfigMaps := corev1.ConfigMapList{}
	k8sLabels := labels.Set{cmLabelOwnerApp: name}
	if err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace, LabelSelector: k8sLabels.AsSelector()},
		&k8sConfigMaps); err != nil {
		return err
	}

	if len(k8sConfigMaps.Items) == 0 {
		return fmt.Errorf("no found application %s with namespace %s", name, namespace)
	}

	k8sConfigMap := k8sConfigMaps.Items[0]
	for _, item := range k8sConfigMaps.Items {
		if item.Labels[cmLabelRevision] > k8sConfigMap.Labels[cmLabelRevision] {
			k8sConfigMap = item
		}
	}

	release, err := decodeRelease(k8sConfigMap.Data[cmReleaseKey])
	if err != nil {
		return fmt.Errorf("decode application %s from configmap failed: %s", name, err.Error())
	}

	if release == nil {
		return fmt.Errorf("application %s should not be empty data in configmap", name)
	}

	manifests := releaseutil.SplitManifests(release.Manifest)
	for _, content := range manifests {
		if err := helper.DeleteResourceFromYaml(cli, content); err != nil {
			return fmt.Errorf("delete resource by yaml content failed: %v", err.Error())
		}
	}

	for _, item := range k8sConfigMaps.Items {
		if err := deleteConfigMap(cli, namespace, item.Name); err != nil {
			return err
		}
	}

	return nil
}

func createApplication(ctx *resttypes.Context, cli client.Client, namespace, chartDir string, app *types.Application) error {
	chartPath := path.Join(chartDir, app.ChartName, app.ChartVersion)
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		return err
	}

	configMapName := app.Name + ".v1"
	if k8sConfigMap, err := getConfigMap(cli, namespace, configMapName); err != nil && apierrors.IsNotFound(err) == false {
		return err
	} else if k8sConfigMap.Name == configMapName {
		return fmt.Errorf("application %s with namespace %s has been installed", app.Name, namespace)
	}

	rawValues, err := getChartValues(ctx, app)
	if err != nil {
		return err
	}
	if rawValues == nil {
		rawValues = make(map[string]interface{})
	}

	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	options := chartutil.ReleaseOptions{
		Name:      app.Name,
		Namespace: namespace,
		IsInstall: true,
	}
	valuesToRender, err := chartutil.ToRenderValues(chartRequested, rawValues, options, DefaultCapabilities)
	if err != nil {
		return err
	}
	valuesToRender["Release"].(map[string]interface{})["Service"] = "singlecloud"

	rel := &release.Release{
		Name:      app.Name,
		Namespace: namespace,
		Chart:     chartRequested,
		Config:    rawValues,
		Info: &release.Info{
			Status: release.StatusDeployed,
		},
		Version: 1,
	}

	files, err := engine.Render(chartRequested, valuesToRender)
	if err != nil {
		return err
	}

	for k, v := range files {
		if strings.HasSuffix(k, notesFileSuffix) {
			if k == path.Join(app.ChartName, "templates", notesFileSuffix) {
				rel.Info.Notes = v
			}
			delete(files, k)
		}
	}

	releaseManifests := bytes.NewBuffer(nil)
	for fileName, content := range files {
		var contentMap map[string]interface{}
		if err := yaml.Unmarshal([]byte(content), &contentMap); err != nil {
			return fmt.Errorf("unmarshal chart file %s content failed: %s", fileName, err.Error())
		}

		if len(contentMap) == 0 {
			continue
		}

		metadata, ok := contentMap["metadata"].(map[interface{}]interface{})
		if ok == false {
			return fmt.Errorf("invalid chart file %s without metadata", fileName)
		}

		metadata["namespace"] = namespace
		if contentByte, err := yaml.Marshal(contentMap); err != nil {
			return fmt.Errorf("marshal chart file %s content failed: %s", fileName, err.Error())
		} else {
			content = string(contentByte)
		}

		if err := helper.CreateResourceFromYaml(cli, content); err != nil {
			return fmt.Errorf("create resource by file %s failed: %s", fileName, err.Error())
		}

		fmt.Fprintf(releaseManifests, "---\n# Source: %s\n%s\n", fileName, content)
	}

	ts := time.Now()
	rel.Manifest = releaseManifests.String()
	rel.Info.FirstDeployed, rel.Info.LastDeployed = ts, ts
	relStr, err := encodeRelease(rel)
	if err != nil {
		return fmt.Errorf("encode release to string failed: %s", err.Error())
	}

	app.Version = rel.Version
	return cli.Create(context.TODO(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
			Labels: map[string]string{
				cmLabelChart:    app.ChartName,
				cmLabelVersion:  app.ChartVersion,
				cmLabelOwnerApp: app.Name,
				cmLabelRevision: strconv.Itoa(rel.Version),
			},
		},
		Data: map[string]string{cmReleaseKey: relStr},
	})
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
		return nil, fmt.Errorf("install chart %s config is invalid: %v", app.ChartName, err.Error())
	}

	return m, nil
}

func encodeRelease(rls *release.Release) (string, error) {
	b, err := yaml.Marshal(rls)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err = w.Write(b); err != nil {
		return "", err
	}
	w.Close()

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func decodeRelease(data string) (*release.Release, error) {
	if data == "" {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	if len(b) > 2 && bytes.Equal(b[0:3], magicGzip) {
		r, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		b2, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		b = b2
	}

	var rls release.Release
	if err := yaml.Unmarshal(b, &rls); err != nil {
		return nil, err
	}
	return &rls, nil
}
