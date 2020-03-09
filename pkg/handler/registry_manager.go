package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	appv1beta1 "github.com/zdnscloud/application-operator/pkg/apis/app/v1beta1"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/x509"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/config"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const (
	registryAppName      = "registry"
	registryChartName    = "harbor"
	registryChartVersion = "1.1.1"
	httpsScheme          = "https//"
)

type RegistryManager struct {
	clusters *ClusterManager
	ca       x509.Certificate
	chartDir string
}

func newRegistryManager(clusterMgr *ClusterManager, chartDir string, caConfig config.RegistryCAConf) (*RegistryManager, error) {
	ca, err := loadRegistryCA(caConfig)
	if err != nil {
		return nil, err
	}
	return &RegistryManager{
		clusters: clusterMgr,
		chartDir: chartDir,
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

	if err := createSysApplication(ctx, cluster, app, m.chartDir, registryChartName, registryAppName, registry.StorageClass); err != nil {
		return nil, err
	}

	registry.SetID(registryAppName)
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
	if id != registryAppName {
		return nil
	}
	return m.get(ctx)
}

func (m *RegistryManager) get(ctx *restresource.Context) restresource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	k8sAppCRD, err := getApplication(cluster.GetKubeClient(), ZCloudNamespace, registryAppName, true)
	if err != nil {
		log.Warnf("get cluster %s application registry by chart name %s failed %s", cluster.Name, monitorChartName, err.Error())
		return nil
	}

	r, err := genRetrunRegistryFromApplication(k8sAppCRD)
	if err != nil {
		log.Warnf("parse k8s app crd to register failed: %s", err.Error())
		return nil
	}

	return r
}

func (m *RegistryManager) Delete(ctx *restresource.Context) *resterr.APIError {
	if ctx.Resource.GetID() != registryAppName {
		return resterr.NewAPIError(resterr.NotFound, "registry doesn't exist")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	if err := deleteApplication(cluster.GetKubeClient(), ZCloudNamespace, registryAppName, true); err != nil {
		return resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("delete application %s failed: %s", registryAppName, err.Error()))
	}

	return nil
}

func genRegistryApplication(cluster *zke.Cluster, registry *types.Registry, ca x509.Certificate) (*types.Application, error) {
	config, err := genRegistryApplicationConfig(cluster, registry, ca)
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

func genRegistryApplicationConfig(cluster *zke.Cluster, registry *types.Registry, ca x509.Certificate) ([]byte, error) {
	domain, err := genIngressDomain(cluster, registry.IngressDomain, registryAppName)
	if err != nil {
		return nil, err
	}
	registry.IngressDomain = domain

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
		ExternalURL:   httpsScheme + registry.IngressDomain,
	}

	return json.Marshal(&harbor)
}

func genRetrunRegistryFromApplication(app *appv1beta1.Application) (*types.Registry, error) {
	h := charts.Harbor{}
	if err := getAppConfigsFromAnnotations(app, &h); err != nil {
		return nil, err
	}

	r := types.Registry{
		IngressDomain: h.Ingress.Core,
		StorageClass:  h.Persistence.StorageClass,
		StorageSize:   h.Persistence.StorageSize,
		RedirectUrl:   httpsScheme + h.Ingress.Core,
		Status:        string(app.Status.State),
	}
	r.SetID(registryAppName)
	r.SetCreationTimestamp(app.CreationTimestamp.Time)
	if app.GetDeletionTimestamp() != nil {
		r.SetDeletionTimestamp(app.DeletionTimestamp.Time)
	}
	return &r, nil
}
