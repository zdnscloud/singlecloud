package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var (
	Version = resttypes.APIVersion{
		Version: "v1",
		Group:   "zcloud.cn",
	}
)

type App struct {
	clusterManager *ClusterManager
}

func NewApp() *App {
	return &App{
		clusterManager: newClusterManager(),
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
	schemas := resttypes.NewSchemas()
	schemas.MustImportAndCustomize(&Version, types.Cluster{}, a.clusterManager, types.SetClusterSchema)
	schemas.MustImportAndCustomize(&Version, types.Node{}, newNodeManager(a.clusterManager), types.SetNodeSchema)
	schemas.MustImportAndCustomize(&Version, types.Namespace{}, newNamespaceManager(a.clusterManager), types.SetNamespaceSchema)
	schemas.MustImportAndCustomize(&Version, types.Deployment{}, newDeploymentManager(a.clusterManager), types.SetDeploymentSchema)
	schemas.MustImportAndCustomize(&Version, types.ConfigMap{}, newConfigMapManager(a.clusterManager), types.SetConfigMapSchema)
	schemas.MustImportAndCustomize(&Version, types.Service{}, newServiceManager(a.clusterManager), types.SetServiceSchema)
	schemas.MustImportAndCustomize(&Version, types.Ingress{}, newIngressManager(a.clusterManager), types.SetIngressSchema)
	schemas.MustImportAndCustomize(&Version, types.Pod{}, newPodManager(a.clusterManager), types.SetPodSchema)
	schemas.MustImportAndCustomize(&Version, types.Job{}, newJobManager(a.clusterManager), types.SetJobSchema)
	schemas.MustImportAndCustomize(&Version, types.CronJob{}, newCronJobManager(a.clusterManager), types.SetCronJobSchema)
	schemas.MustImportAndCustomize(&Version, types.DaemonSet{}, newDaemonSetManager(a.clusterManager), types.SetDaemonSetSchema)
	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		return err
	}
	adaptor.RegisterHandler(router, gin.WrapH(server), server.Schemas.UrlMethods())
	return nil
}

const (
	WSPrefix         = "/apis/ws.zcloud.cn/v1"
	WSShellPathTemp  = WSPrefix + "/clusters/%s/shell"
	WSPodLogPathTemp = WSPrefix + "/clusters/%s/namespaces/%s/pods/%s/containers/%s/log"
)

func (a *App) registerWSHandler(router gin.IRoutes) {
	shellPath := fmt.Sprintf(WSShellPathTemp, ":cluster") + "/*actions"
	router.GET(shellPath, func(c *gin.Context) {
		a.clusterManager.OpenConsole(c.Param("cluster"), c.Request, c.Writer)
	})

	podLogPath := fmt.Sprintf(WSPodLogPathTemp, ":cluster", ":namespace", ":pod", ":container") + "/*actions"
	router.GET(podLogPath, func(c *gin.Context) {
		a.clusterManager.OpenPodLog(c.Param("cluster"), c.Param("namespace"), c.Param("pod"), c.Param("container"), c.Request, c.Writer)
	})
}