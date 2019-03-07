package types

import (
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/exec"
	"github.com/zdnscloud/gorest/types"
)

func SetClusterSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET"}
}

type Cluster struct {
	types.Resource `json:",inline"`
	Name           string `json:"name,omitempty"`
	NodesCount     uint32 `json:"nodeCount,omitempty"`
	Version        string `json:"version,omitempty"`

	KubeClient client.Client  `json:"-"`
	Executor   *exec.Executor `json:"-"`
}

var ClusterType = types.GetResourceType(Cluster{})
