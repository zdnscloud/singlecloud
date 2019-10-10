package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/gorest"
	"github.com/zdnscloud/gorest/adaptor"
	restresource "github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/gorest/resource/schema"
	"github.com/zdnscloud/singlecloud/pkg/authentication"
	"github.com/zdnscloud/singlecloud/pkg/authorization"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
	"github.com/zdnscloud/singlecloud/storage"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "zcloud.cn",
	}
)

type App struct {
	clusterManager *ClusterManager
	chartDir       string
	repoUrl        string
}

func NewApp(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, eventBus *pubsub.PubSub, agent *clusteragent.AgentManager, db storage.DB, chartDir, scVersion, repoUrl string) (*App, error) {
	clusterMgr, err := newClusterManager(authenticator, authorizer, eventBus, agent, db, scVersion)
	if err != nil {
		return nil, err
	}
	return &App{
		clusterManager: clusterMgr,
		chartDir:       chartDir,
		repoUrl:        repoUrl,
	}, nil
}

func (a *App) RegisterHandler(router gin.IRoutes) error {
	if err := a.registerRestHandler(router); err != nil {
		return err
	}
	a.registerWSHandler(router)
	return nil
}

func (a *App) registerRestHandler(router gin.IRoutes) error {
	schemas := schema.NewSchemaManager()
	schemas.MustImport(&Version, types.Cluster{}, a.clusterManager)
	schemas.MustImport(&Version, types.Node{}, newNodeManager(a.clusterManager))
	schemas.MustImport(&Version, types.PodNetwork{}, newPodNetworkManager(a.clusterManager))
	schemas.MustImport(&Version, types.NodeNetwork{}, newNodeNetworkManager(a.clusterManager))
	schemas.MustImport(&Version, types.ServiceNetwork{}, newServiceNetworkManager(a.clusterManager))
	schemas.MustImport(&Version, types.BlockDevice{}, newBlockDeviceManager(a.clusterManager))
	schemas.MustImport(&Version, types.StorageCluster{}, newStorageClusterManager(a.clusterManager))
	schemas.MustImport(&Version, types.Namespace{}, newNamespaceManager(a.clusterManager))
	schemas.MustImport(&Version, types.Chart{}, newChartManager(a.chartDir, a.repoUrl))
	schemas.MustImport(&Version, types.ConfigMap{}, newConfigMapManager(a.clusterManager))
	schemas.MustImport(&Version, types.CronJob{}, newCronJobManager(a.clusterManager))
	schemas.MustImport(&Version, types.DaemonSet{}, newDaemonSetManager(a.clusterManager))
	schemas.MustImport(&Version, types.Deployment{}, newDeploymentManager(a.clusterManager))
	schemas.MustImport(&Version, types.Ingress{}, newIngressManager(a.clusterManager))
	schemas.MustImport(&Version, types.Job{}, newJobManager(a.clusterManager))
	schemas.MustImport(&Version, types.LimitRange{}, newLimitRangeManager(a.clusterManager))
	schemas.MustImport(&Version, types.PersistentVolumeClaim{}, newPersistentVolumeClaimManager(a.clusterManager))
	schemas.MustImport(&Version, types.PersistentVolume{}, newPersistentVolumeManager(a.clusterManager))
	schemas.MustImport(&Version, types.ResourceQuota{}, newResourceQuotaManager(a.clusterManager))
	schemas.MustImport(&Version, types.Secret{}, newSecretManager(a.clusterManager))
	schemas.MustImport(&Version, types.Service{}, newServiceManager(a.clusterManager))
	schemas.MustImport(&Version, types.StatefulSet{}, newStatefulSetManager(a.clusterManager))
	schemas.MustImport(&Version, types.Pod{}, newPodManager(a.clusterManager))
	schemas.MustImport(&Version, types.UdpIngress{}, newUDPIngressManager(a.clusterManager))
	schemas.MustImport(&Version, types.UserQuota{}, newUserQuotaManager(a.clusterManager))
	schemas.MustImport(&Version, types.StorageClass{}, newStorageClassManager(a.clusterManager))
	schemas.MustImport(&Version, types.InnerService{}, newInnerServiceManager(a.clusterManager))
	schemas.MustImport(&Version, types.OuterService{}, newOuterServiceManager(a.clusterManager))
	schemas.MustImport(&Version, types.KubeConfig{}, newKubeConfigManager(a.clusterManager))

	appManager := newApplicationManager(a.clusterManager, a.chartDir)
	if err := appManager.addChartsConfig(charts.SupportChartsConfig); err != nil {
		return err
	}
	schemas.MustImport(&Version, types.Application{}, appManager)
	schemas.MustImport(&Version, types.Monitor{}, newMonitorManager(a.clusterManager, appManager))
	schemas.MustImport(&Version, types.Registry{}, newRegistryManager(a.clusterManager, appManager))

	userManager := newUserManager(a.clusterManager.authenticator.JwtAuth, a.clusterManager.authorizer)
	schemas.MustImport(&Version, types.User{}, userManager)
	server := gorest.NewAPIServer(schemas)
	server.Use(a.clusterManager.authorizationHandler())
	adaptor.RegisterHandler(router, server, schemas.GenerateResourceRoute())
	return nil
}

const (
	WSPrefix         = "/apis/ws.zcloud.cn/v1"
	WSPodLogPathTemp = WSPrefix + "/clusters/%s/namespaces/%s/pods/%s/containers/%s/log"
)

func (a *App) registerWSHandler(router gin.IRoutes) {
	podLogPath := fmt.Sprintf(WSPodLogPathTemp, ":cluster", ":namespace", ":pod", ":container") + "/*actions"
	router.GET(podLogPath, func(c *gin.Context) {
		a.clusterManager.OpenPodLog(c.Param("cluster"), c.Param("namespace"), c.Param("pod"), c.Param("container"), c.Request, c.Writer)
	})

	zkeLogPath := fmt.Sprintf(zke.WSZKELogPathTemp, ":cluster") + "/*actions"
	router.GET(zkeLogPath, func(c *gin.Context) {
		a.clusterManager.zkeManager.OpenLog(c.Param("cluster"), c.Request, c.Writer)
	})
}
