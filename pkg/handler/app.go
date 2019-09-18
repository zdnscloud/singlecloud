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
}

func NewApp(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, eventBus *pubsub.PubSub, agent *clusteragent.AgentManager, db storage.DB, chartDir string) *App {
	return &App{
		clusterManager: newClusterManager(authenticator, authorizer, eventBus, agent, db),
		chartDir:       chartDir,
	}
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
	schemas.Import(&Version, types.Cluster{}, a.clusterManager)
	schemas.Import(&Version, types.Node{}, newNodeManager(a.clusterManager))
	schemas.Import(&Version, types.Namespace{}, newNamespaceManager(a.clusterManager))
	schemas.Import(&Version, types.Chart{}, newChartManager(a.chartDir))
	schemas.Import(&Version, types.ConfigMap{}, newConfigMapManager(a.clusterManager))
	schemas.Import(&Version, types.CronJob{}, newCronJobManager(a.clusterManager))
	schemas.Import(&Version, types.DaemonSet{}, newDaemonSetManager(a.clusterManager))
	schemas.Import(&Version, types.Deployment{}, newDeploymentManager(a.clusterManager))
	schemas.Import(&Version, types.Ingress{}, newIngressManager(a.clusterManager))
	schemas.Import(&Version, types.Job{}, newJobManager(a.clusterManager))
	schemas.Import(&Version, types.LimitRange{}, newLimitRangeManager(a.clusterManager))
	schemas.Import(&Version, types.NodeNetwork{}, newNodeNetworkManager(a.clusterManager))
	schemas.Import(&Version, types.Pod{}, newPodManager(a.clusterManager))
	schemas.Import(&Version, types.PodNetwork{}, newPodNetworkManager(a.clusterManager))
	schemas.Import(&Version, types.PersistentVolumeClaim{}, newPersistentVolumeClaimManager(a.clusterManager))
	schemas.Import(&Version, types.PersistentVolume{}, newPersistentVolumeManager(a.clusterManager))
	schemas.Import(&Version, types.ResourceQuota{}, newResourceQuotaManager(a.clusterManager))
	schemas.Import(&Version, types.Secret{}, newSecretManager(a.clusterManager))
	schemas.Import(&Version, types.Service{}, newServiceManager(a.clusterManager))
	schemas.Import(&Version, types.ServiceNetwork{}, newServiceNetworkManager(a.clusterManager))
	schemas.Import(&Version, types.StatefulSet{}, newStatefulSetManager(a.clusterManager))
	schemas.Import(&Version, types.UdpIngress{}, newUDPIngressManager(a.clusterManager))
	schemas.Import(&Version, types.UserQuota{}, newUserQuotaManager(a.clusterManager))

	appManager := newApplicationManager(a.clusterManager, a.chartDir)
	if err := appManager.addChartsConfig(charts.SupportChartsConfig); err != nil {
		return err
	}
	schemas.Import(&Version, types.Application{}, appManager)

	userManager := newUserManager(a.clusterManager.authenticator.JwtAuth, a.clusterManager.authorizer)
	schemas.Import(&Version, types.User{}, userManager)
	server := gorest.NewAPIServer(schemas)
	server.Use(a.clusterManager.authorizationHandler())
	adaptor.RegisterHandler(router, gorest.NewAPIServer(schemas), schemas.GenerateResourceRoute())
	return nil
}

const (
	WSPrefix         = "/apis/ws.zcloud.cn/v1"
	WSPodLogPathTemp = WSPrefix + "/clusters/%s/namespaces/%s/pods/%s/containers/%s/log"
)

func (a *App) registerWSHandler(router gin.IRoutes) {
	// podLogPath := fmt.Sprintf(WSPodLogPathTemp, ":cluster", ":namespace", ":pod", ":container") + "/*actions"
	// router.GET(podLogPath, func(c *gin.Context) {
	// a.clusterManager.OpenPodLog(c.Param("cluster"), c.Param("namespace"), c.Param("pod"), c.Param("container"), c.Request, c.Writer)
	// })

	zkeLogPath := fmt.Sprintf(zke.WSZKELogPathTemp, ":cluster") + "/*actions"
	router.GET(zkeLogPath, func(c *gin.Context) {
		a.clusterManager.zkeManager.OpenLog(c.Param("cluster"), c.Request, c.Writer)
	})
}
