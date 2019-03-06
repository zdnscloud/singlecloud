package k8smanager

import (
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var (
	version = resttypes.APIVersion{
		Version: "v1",
		Group:   "zcloud.cn",
		Path:    "/v1",
	}
)

func NewRestHandler() (*api.Server, *Handler, error) {
	server := api.NewAPIServer()
	restAPIHandler := NewHandler()
	schemas := resttypes.NewSchemas()
	schemas.MustImportAndCustomize(&version, types.Cluster{}, restAPIHandler, types.SetClusterSchema)
	schemas.MustImportAndCustomize(&version, types.Node{}, restAPIHandler, types.SetNodeSchema)
	schemas.MustImportAndCustomize(&version, types.Namespace{}, restAPIHandler, types.SetNamespaceSchema)
	schemas.MustImportAndCustomize(&version, types.Deployment{}, restAPIHandler, types.SetDeploymentSchema)
	if err := server.AddSchemas(schemas); err != nil {
		return nil, nil, err
	}
	return server, restAPIHandler, nil
}
