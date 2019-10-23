package handler

import (
	"encoding/json"
	"fmt"
	"github.com/zdnscloud/singlecloud/pkg/zke"
	"strconv"
	"strings"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/randomdata"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"
)

const (
	efkChartName       = "efk"
	efkChartVersion    = "0.0.1"
	efkAppNamePrefix   = "efk"
	efkStorageSizeUint = "Gi"
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
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	efk := ctx.Resource.(*types.EFK)
	app, err := genEFKApplication(cluster, efk)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, err.Error())
	}

	if err := createSysApplication(ctx, m.clusters.GetDB(), m.apps, cluster, efkChartName, app, efk.StorageClass, efkAppNamePrefix); err != nil {
		return nil, err
	}

	efk.Status = types.AppStatusCreate
	efk.SetID(efkAppNamePrefix)
	return efk, nil
}

func (m *EFKManager) Delete(ctx *restresource.Context) *resterr.APIError {
	if ctx.Resource.GetID() != efkAppNamePrefix {
		return resterr.NewAPIError(resterr.NotFound, "efk doesn't exist")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	return deleteApplicationByChartName(m.clusters.GetDB(), cluster, efkChartName)
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

	app, err := getApplicationFromDBByChartName(m.clusters.GetDB(), storage.GenTableName(ApplicationTable, cluster.Name, ZCloudNamespace), efkChartName)
	if err != nil {
		log.Warnf("get cluster %s application by chart name %s failed %s", cluster.Name, efkChartName, err.Error())
		return nil
	}
	if app == nil {
		return nil
	}

	efk, err := genEFKFromApp(ctx, cluster.Name, app)
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

func genEFKApplication(cluster *zke.Cluster, efk *types.EFK) (*types.Application, error) {
	config, err := genEFKConfigs(cluster, efk)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         efkAppNamePrefix + "-" + randomdata.RandString(12),
		ChartName:    efkChartName,
		ChartVersion: efkChartVersion,
		Configs:      config,
		SystemChart:  true,
	}, nil
}

func genEFKConfigs(cluster *zke.Cluster, efk *types.EFK) ([]byte, error) {
	if len(efk.IngressDomain) == 0 {
		edgeIPs := cluster.GetNodeIpsByRole(types.RoleEdge)
		if len(edgeIPs) == 0 {
			return nil, fmt.Errorf("can not find edge node for this cluster")
		}
		efk.IngressDomain = efkAppNamePrefix + "-" + ZCloudNamespace + "-" + cluster.Name + "." + edgeIPs[0] + "." + ZcloudDynamicaDomainPrefix
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
	return json.Marshal(&e)
}
