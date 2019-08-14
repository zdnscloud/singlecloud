package handler

import (
	"encoding/json"
	"fmt"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	registryNameSpace    = "registry"
	registryAppName      = "zcloud-registry"
	registryChartName    = "harbor"
	registryChartVersion = "v1.1.1"
)

type RegistryManager struct {
	api.DefaultHandler
	clusters *ClusterManager
	apps     *ApplicationManager
}

func newRegistryManager(clusterMgr *ClusterManager, appMgr *ApplicationManager) *RegistryManager {
	return &RegistryManager{
		clusters: clusterMgr,
		apps:     appMgr,
	}
}

func (m *RegistryManager) Create(ctx *resttypes.Context, yaml []byte) (interface{}, *resttypes.APIError) {
	r := ctx.Object.(*types.Registry)
	url := genUrlPrefix(ctx, r.Cluster, registryNameSpace)
	cluster := m.clusters.GetClusterByName(r.Cluster)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}
	app, err := genRegistryApplication(r)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	app.SetID(app.Name)
	if err := m.apps.create(ctx, cluster.KubeClient, registryNameSpace, url, app); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("create registry application failed %s", err.Error()))
	}
	r.SetID(app.Name)
	return r, nil
}

func (m *RegistryManager) Get(ctx *resttypes.Context) interface{} {
	return nil
}

func (m *RegistryManager) List(ctx *resttypes.Context) interface{} {
	rs := []*types.Registry{}
	r := &types.Registry{
		Name:          "test-registry",
		Cluster:       "wang",
		IngressDomain: "core.harbor.cluster.w",
		StorageClass:  "lvm",
		StorageSize:   "10Gi",
		AdminPassword: "Harbor1234",
		RedirectUrl:   "http://core.harbor.cluster.w",
	}
	rs = append(rs, r)
	return rs
}

func (m *RegistryManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	return nil
}

func genRegistryApplication(r *types.Registry) (*types.Application, error) {
	config, err := genRegistryConfigs(r)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         registryAppName,
		ChartName:    registryChartName,
		ChartVersion: registryChartVersion,
		Configs:      config,
	}, nil
}

func genRegistryConfigs(r *types.Registry) (string, error) {
	harbor := charts.Harbor{
		IngressDomain:       r.IngressDomain,
		StorageClass:        r.StorageClass,
		RegistryStorageSize: r.StorageSize,
		AdminPassword:       r.AdminPassword,
	}
	content, err := json.Marshal(&harbor)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
