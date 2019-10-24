package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/zdnscloud/gok8s/client"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"
)

const (
	efkChartName           = "efk"
	efkChartVersion        = "0.0.1"
	efkAppNamePrefix       = "efk"
	efkStorageSizeUint     = "Gi"
	defaultEFKESReplicas   = 3
	defaultEFKStorageClass = "lvm"
	defaultEFKStorageSize  = 10
)

type EFKManager struct {
	clusters *ClusterManager
	apps     *ApplicationManager
}

func newEFKManager(clusterMgr *ClusterManager, appMgr *ApplicationManager) *EFKManager {
	return &EFKManager{
		clusters: clusterMgr,
		apps:     appMgr,
	}
}

func (m *EFKManager) Create(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can create cluster efk")
	}
	efk := ctx.Resource.(*types.EFK)
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	existApps, err := getApplicationsFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), efkChartName)

	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get efk application from db failed %s", err.Error()))
	}
	if len(existApps) > 0 {
		return nil, resterr.NewAPIError(resterr.DuplicateResource, fmt.Sprintf("cluster %s efk has exist", cluster.Name))
	}

	// check the storage class exist
	requiredStorageClass := defaultEFKStorageClass
	if len(efk.StorageClass) > 0 {
		requiredStorageClass = efk.StorageClass
	}
	if !isStorageClassExist(cluster.KubeClient, requiredStorageClass) {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("%s storageclass does't exist in cluster %s", requiredStorageClass, cluster.Name))
	}

	if efk.StorageSize == 0 {
		efk.StorageSize = defaultEFKStorageSize
	}
	if efk.ESReplicas == 0 {
		efk.ESReplicas = defaultEFKESReplicas
	}

	app, err := genEFKApplication(cluster.KubeClient, efk, cluster.Name)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, err.Error())
	}
	tableName := storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace)
	app.Name = genAppNameIfDuplicate(m.clusters.GetDB(), tableName, app.Name, efkAppNamePrefix)

	app.SetID(app.Name)
	if err := m.apps.create(ctx, cluster, ZCloudNamespace, app); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("create efk application failed %s", err.Error()))
	}

	go ensureApplicationSucceedOrDie(ctx, m.clusters.GetDB(), cluster, tableName, app.Name)

	efk.Status = types.AppStatusCreate
	efk.SetID(efkAppNamePrefix)
	efk.SetCreationTimestamp(time.Now())
	return efk, nil
}

func (m *EFKManager) Delete(ctx *restresource.Context) *resterr.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	apps, err := getApplicationsFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), efkChartName)
	if err != nil || len(apps) == 0 {
		return resterr.NewAPIError(resterr.NotFound, "efk doesn't exist")
	}

	appName := apps[0].Name
	tableName := storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace)
	app, err := updateAppStatusToDeleteFromDB(m.clusters.GetDB(), tableName, appName, true)
	if err != nil {
		if err == storage.ErrNotFoundResource {
			return resterr.NewAPIError(resterr.NotFound,
				fmt.Sprintf("application %s with namespace %s doesn't exist", appName, ZCloudNamespace))
		} else {
			return resterr.NewAPIError(resterr.ServerError,
				fmt.Sprintf("delete application %s failed: %s", appName, err.Error()))
		}
	}

	go deleteApplication(m.clusters.GetDB(), cluster.KubeClient, tableName, ZCloudNamespace, app)
	return nil
}

func (m *EFKManager) List(ctx *restresource.Context) interface{} {
	efk := m.get(ctx)
	if efk == nil {
		return nil
	} else {
		return []*types.EFK{efk.(*types.EFK)}
	}
}

func (m *EFKManager) Get(ctx *restresource.Context) restresource.Resource {
	id := ctx.Resource.GetID()
	if id != efkAppNamePrefix {
		return nil
	}
	return m.get(ctx)
}

func (m *EFKManager) get(ctx *restresource.Context) restresource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	apps, err := getApplicationsFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), efkChartName)
	if err != nil || len(apps) == 0 {
		return nil
	}

	efk, err := genEFKFromApp(ctx, cluster.Name, apps[0])
	if err != nil {
		return nil
	}

	return efk
}

func genEFKFromApp(ctx *restresource.Context, cluster string, app *types.Application) (*types.EFK, error) {
	e := charts.EFK{}
	if err := json.Unmarshal(app.Configs, &e); err != nil {
		return nil, err
	}
	sizeString := strings.Split(e.Elasticsearch.VolumeClaimTemplate.Resources.Requests.Storage, efkStorageSizeUint)[0]
	size, _ := strconv.Atoi(sizeString)
	efk := types.EFK{
		IngressDomain: e.Kibana.Ingress.Hosts,
		ESReplicas:    e.Elasticsearch.Replicas,
		StorageClass:  e.Elasticsearch.VolumeClaimTemplate.StorageClass,
		StorageSize:   size,
		RedirectUrl:   "http://" + e.Kibana.Ingress.Hosts,
		Status:        app.Status,
	}
	efk.SetID(efkAppNamePrefix)
	efk.CreationTimestamp = app.CreationTimestamp
	return &efk, nil
}

func genEFKApplication(cli client.Client, efk *types.EFK, clusterName string) (*types.Application, error) {
	config, err := genEFKConfigs(cli, efk, clusterName)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         efkAppNamePrefix + "-" + genRandomStr(12),
		ChartName:    efkChartName,
		ChartVersion: efkChartVersion,
		Configs:      config,
		SystemChart:  true,
	}, nil
}

func genEFKConfigs(cli client.Client, efk *types.EFK, clusterName string) ([]byte, error) {
	if len(efk.IngressDomain) == 0 {
		edgeNodeIP := randomEdgeNodeAddress(cli)
		if len(edgeNodeIP) == 0 {
			return nil, fmt.Errorf("can not find edge node for this cluster")
		}
		efk.IngressDomain = efkAppNamePrefix + "-" + ZCloudNamespace + "-" + clusterName + "." + edgeNodeIP + "." + ZcloudDynamicaDomainPrefix
	}
	size := strconv.Itoa(efk.StorageSize) + efkStorageSizeUint
	efk.RedirectUrl = "http://" + efk.IngressDomain
	e := charts.EFK{
		Elasticsearch: charts.ES{
			Replicas: efk.ESReplicas,
			VolumeClaimTemplate: charts.Pvc{
				StorageClass: efk.StorageClass,
				Resources: charts.PvcResources{
					Requests: charts.PvcRequests{
						Storage: size,
					},
				},
			},
		},
		Kibana: charts.KA{
			Ingress: charts.KibanaIngress{
				Hosts: efk.IngressDomain,
			},
		},
	}
	content, err := json.Marshal(&e)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func randomEdgeNodeAddress(cli client.Client) string {
	nodes, err := getNodes(cli)
	if err != nil {
		return ""
	}
	var ips []string
	for _, n := range nodes {
		if !n.HasRole(types.RoleEdge) {
			continue
		}
		ips = append(ips, n.Address)
	}
	if len(ips) > 0 {
		rand.Seed(time.Now().UnixNano())
		return ips[rand.Intn(len(ips))]
	} else {
		return ""
	}
}
