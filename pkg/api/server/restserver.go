package server

import (
	"github.com/zdnscloud/gorest/api"
	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/handler"
	"github.com/zdnscloud/singlecloud/pkg/types/cluster"
	"github.com/zdnscloud/singlecloud/pkg/types/node"
)

var (
	version = types.APIVersion{
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
	schemas := types.NewSchemas()
	schemas.MustImportAndCustomize(&version, cluster.Cluster{}, &handler.Handler{}, cluster.SetSchema)
	schemas.MustImportAndCustomize(&version, node.Node{}, &handler.Handler{}, node.SetSchema)
	if err := server.AddSchemas(schemas); err != nil {
		return nil, err
	}

	return &RestServer{
		server: server,
	}, nil
}
