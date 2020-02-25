package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	appv1beta1 "github.com/zdnscloud/application-operator/pkg/apis/app/v1beta1"
	"github.com/zdnscloud/cement/log"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const (
	efkChartName    = "efk"
	efkChartVersion = "0.0.1"
	efkAppName      = "efk"
)

type EFKManager struct {
	clusters *ClusterManager
	chartDir string
}

func newEFKManager(clusterMgr *ClusterManager, chartDir string) *EFKManager {
	return &EFKManager{
		clusters: clusterMgr,
		chartDir: chartDir,
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

	if err := createSysApplication(ctx, cluster, app, m.chartDir, efkChartName, efkAppName, efk.StorageClass); err != nil {
		return nil, err
	}

	efk.SetID(efkAppName)
	return efk, nil
}

func (m *EFKManager) Delete(ctx *restresource.Context) *resterr.APIError {
	if ctx.Resource.GetID() != efkAppName {
		return resterr.NewAPIError(resterr.NotFound, "efk doesn't exist")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	if err := deleteApplication(cluster.GetKubeClient(), ZCloudNamespace, efkAppName, true); err != nil {
		return resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("delete application %s failed: %s", efkAppName, err.Error()))
	}

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
	if id != efkAppName {
		return nil
	}
	return m.get(ctx)
}

func (m *EFKManager) get(ctx *restresource.Context) restresource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	k8sAppCRD, err := getApplication(cluster.GetKubeClient(), ZCloudNamespace, efkAppName, true)
	if err != nil {
		log.Warnf("get cluster %s application efk by chart name %s failed %s", cluster.Name, monitorChartName, err.Error())
		return nil
	}

	efk, err := genEFKFromApp(k8sAppCRD)
	if err != nil {
		return nil
	}
	return efk
}

func genEFKFromApp(app *appv1beta1.Application) (*types.EFK, error) {
	e := charts.EFK{}
	if err := getAppConfigsFromAnnotations(app, &e); err != nil {
		return nil, err
	}

	efk := types.EFK{
		IngressDomain: e.Kibana.Ingress.Hosts,
		ESReplicas:    e.Elasticsearch.Replicas,
		StorageClass:  e.Elasticsearch.VolumeClaimTemplate.StorageClass,
		StorageSize:   e.Elasticsearch.VolumeClaimTemplate.Resources.Requests.Storage,
		RedirectUrl:   "http://" + e.Kibana.Ingress.Hosts,
		Status:        string(app.Status.State),
	}
	efk.SetID(efkAppName)
	efk.SetCreationTimestamp(app.CreationTimestamp.Time)
	if app.GetDeletionTimestamp() != nil {
		efk.SetDeletionTimestamp(app.DeletionTimestamp.Time)
	}
	return &efk, nil
}

func genEFKApplication(cluster *zke.Cluster, efk *types.EFK) (*types.Application, error) {
	config, err := genEFKConfigs(cluster, efk)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         efkAppName,
		ChartName:    efkChartName,
		ChartVersion: efkChartVersion,
		Configs:      config,
	}, nil
}

func genEFKConfigs(cluster *zke.Cluster, efk *types.EFK) ([]byte, error) {
	domain, err := genIngressDomain(cluster, efk.IngressDomain, efkAppName)
	if err != nil {
		return nil, err
	}
	efk.IngressDomain = domain

	e := charts.EFK{
		Elasticsearch: charts.ES{
			Replicas: efk.ESReplicas,
			VolumeClaimTemplate: charts.Pvc{
				StorageClass: efk.StorageClass,
				Resources: charts.PvcResources{
					Requests: charts.PvcRequests{
						Storage: efk.StorageSize / types.DefaultEFKESReplicas,
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

func getRandomEdgeNodeAddress(cluster *zke.Cluster) string {
	ips := cluster.GetNodeIpsByRole(types.RoleEdge)
	if len(ips) > 0 {
		rand.Seed(time.Now().UnixNano())
		return ips[rand.Intn(len(ips))]
	}
	return ""
}
