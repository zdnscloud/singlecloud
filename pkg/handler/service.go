package handler

import (
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var (
	Version = resttypes.APIVersion{
		Version: "v1",
		Group:   "zcloud.cn",
		Path:    "/v1",
	}
)

func NewRestHandler() (*api.Server, *ClusterManager, error) {
	schemas := resttypes.NewSchemas()
	clusterHandler := newClusterManager()
	schemas.MustImportAndCustomize(&Version, types.Cluster{}, clusterHandler, types.SetClusterSchema)
	schemas.MustImportAndCustomize(&Version, types.Node{}, newNodeManager(clusterHandler), types.SetNodeSchema)
	schemas.MustImportAndCustomize(&Version, types.Namespace{}, newNamespaceManager(clusterHandler), types.SetNamespaceSchema)
	schemas.MustImportAndCustomize(&Version, types.Deployment{}, newDeploymentManager(clusterHandler), types.SetDeploymentSchema)
	schemas.MustImportAndCustomize(&Version, types.ConfigMap{}, newConfigMapManager(clusterHandler), types.SetConfigMapSchema)

	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		return nil, nil, err
	}
	return server, clusterHandler, nil
}
