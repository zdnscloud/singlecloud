package server

import (
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/handler"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

var (
	version = resttypes.APIVersion{
		Version: "v1",
		Group:   "zcloud.cn",
		Path:    "/v1",
	}
)

type RestServer struct {
	server *api.Server
}

func newRestServer() (*RestServer, error) {
	server := api.NewAPIServer()
	restAPIHandler := handler.NewHandler()
	schemas := resttypes.NewSchemas()
	schemas.MustImportAndCustomize(&version, types.Cluster{}, restAPIHandler, types.SetClusterSchema)
	schemas.MustImportAndCustomize(&version, types.Node{}, restAPIHandler, types.SetNodeSchema)
	if err := server.AddSchemas(schemas); err != nil {
		return nil, err
	}

	return &RestServer{
		server: server,
	}, nil
}
