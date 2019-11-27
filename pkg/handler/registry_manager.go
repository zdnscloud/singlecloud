package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/zdnscloud/singlecloud/config"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/randomdata"
	"github.com/zdnscloud/cement/x509"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
)

const (
	registryAppNamePrefix = "registry"
	registryChartName     = "harbor"
	registryChartVersion  = "1.1.1"
)

type RegistryManager struct {
	clusters *ClusterManager
	apps     *ApplicationManager
	ca       x509.Certificate
}

func newRegistryManager(clusterMgr *ClusterManager, appMgr *ApplicationManager, caConfig config.RegistryCAConf) (*RegistryManager, error) {
	ca, err := loadRegistryCA(caConfig)
	if err != nil {
		return nil, err
	}
	return &RegistryManager{
		clusters: clusterMgr,
		apps:     appMgr,
		ca:       ca,
	}, nil
}

func loadRegistryCA(caConfig config.RegistryCAConf) (x509.Certificate, error) {
	ca := x509.Certificate{}
	cert, err := ioutil.ReadFile(caConfig.CaCertPath)
	if err != nil {
		return ca, fmt.Errorf("load registry ca failed %s", err.Error())
	}
	key, err := ioutil.ReadFile(caConfig.CaKeyPath)
	if err != nil {
		return ca, fmt.Errorf("load registry ca failed %s", err.Error())
	}
	ca.Cert = string(cert)
	ca.Key = string(key)

	if _, err = x509.GenerateSignedCertificate("test.registry.zdns.cn", nil, []interface{}{"test.registry.zdns.cn"}, 7300, ca); err != nil {
		return ca, fmt.Errorf("verify registry ca failed %s", err.Error())
	}
	return ca, nil
}

func (m *RegistryManager) Create(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can create registry")
	}

	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	registry := ctx.Resource.(*types.Registry)
	app, err := genRegistryApplication(cluster, registry, m.ca)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, err.Error())
	}

	if err := createSysApplication(ctx, m.clusters.GetDB(), m.apps, cluster, registryChartName, app, registry.StorageClass, registryAppNamePrefix); err != nil {
		return nil, err
	}

	registry.Status = types.AppStatusCreate
	registry.SetID(registryAppNamePrefix)
	return registry, nil
}

func (m *RegistryManager) List(ctx *restresource.Context) interface{} {
	r := m.get(ctx)
	if r == nil {
		return nil
	} else {
		return []*types.Registry{r.(*types.Registry)}
	}
}

func (m *RegistryManager) Get(ctx *restresource.Context) restresource.Resource {
	id := ctx.Resource.GetID()
	if id != registryAppNamePrefix {
		return nil
	}
	return m.get(ctx)
}

func (m *RegistryManager) get(ctx *restresource.Context) restresource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	app, err := getApplicationFromDBByChartName(m.clusters.GetDB(), cluster.Name, registryChartName)
	if err != nil {
		log.Warnf("get cluster %s application by chart name %s failed %s", cluster.Name, registryChartName, err.Error())
		return nil
	}
	if app == nil {
		return nil
	}

	r, err := genRetrunRegistryFromApplication(cluster.Name, app)
	if err != nil {
		return nil
	}

	return r
}

func (m *RegistryManager) Delete(ctx *restresource.Context) *resterr.APIError {
	if ctx.Resource.GetID() != registryAppNamePrefix {
		return resterr.NewAPIError(resterr.NotFound, "registry doesn't exist")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	return deleteApplicationByChartName(m.clusters.GetDB(), cluster, registryChartName)
}

func genRegistryApplication(cluster *zke.Cluster, registry *types.Registry, ca x509.Certificate) (*types.Application, error) {
	config, err := genRegistryApplicationConfig(cluster, registry, ca)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         registryAppNamePrefix + "-" + randomdata.RandString(12),
		ChartName:    registryChartName,
		ChartVersion: registryChartVersion,
		Configs:      config,
		SystemChart:  true,
	}, nil
}

func genRegistryApplicationConfig(cluster *zke.Cluster, registry *types.Registry, ca x509.Certificate) ([]byte, error) {
	if len(registry.IngressDomain) == 0 {
		edgeIP := getRandomEdgeNodeAddress(cluster)
		if len(edgeIP) == 0 {
			return nil, fmt.Errorf("can not find edge node for this cluster")
		}
		registry.IngressDomain = registryAppNamePrefix + "-" + ZCloudNamespace + "-" + cluster.Name + "." + edgeIP + "." + ZcloudDynamicaDomainPrefix
	}
	registry.RedirectUrl = "https://" + registry.IngressDomain

	tls, err := x509.GenerateSignedCertificate(registry.IngressDomain, nil, []interface{}{registry.IngressDomain}, 7300, ca)
	if err != nil {
		return nil, err
	}

	harbor := charts.Harbor{
		Ingress: charts.HarborIngress{
			Core:   registry.IngressDomain,
			CaCrt:  ca.Cert,
			TlsCrt: tls.Cert,
			TlsKey: tls.Key,
		},
		Persistence: charts.HarborPersistence{
			StorageClass: registry.StorageClass,
			StorageSize:  registry.StorageSize,
		},
		AdminPassword: registry.AdminPassword,
		ExternalURL:   "https://" + registry.IngressDomain,
	}

	return json.Marshal(&harbor)
}

func genRetrunRegistryFromApplication(cluster string, app *types.Application) (*types.Registry, error) {
	h := charts.Harbor{}
	if err := json.Unmarshal(app.Configs, &h); err != nil {
		return nil, err
	}
	r := types.Registry{
		IngressDomain: h.Ingress.Core,
		StorageClass:  h.Persistence.StorageClass,
		StorageSize:   h.Persistence.StorageSize,
		RedirectUrl:   "https://" + h.Ingress.Core,
		Status:        app.Status,
	}
	r.SetID(registryAppNamePrefix)
	r.CreationTimestamp = app.CreationTimestamp
	return &r, nil
}
