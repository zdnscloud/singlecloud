package k8smanager

import (
	"fmt"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type ClusterManager struct {
	clusters []*types.Cluster
}

func newClusterManager() *ClusterManager {
	return &ClusterManager{}
}

func (m *ClusterManager) Create(cluster *types.Cluster, yamlConf []byte) (*types.Cluster, *resttypes.APIError) {
	for _, c := range m.clusters {
		if c.Name == cluster.Name {
			return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate cluster name")
		}
	}

	cluster.SetID(cluster.Name)
	k8sconf, err := config.BuildConfig(yamlConf)
	if err != nil {
		return nil, resttypes.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("invalid cluster config:%s", err.Error()))
	}

	cli, err := client.New(k8sconf, client.Options{})
	if err != nil {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("connect to cluster failed:%s", err.Error()))
	}

	nodes, err := getNodes(cli)
	if err != nil {
		logger.Error("get nodes failed:%s", err.Error())
	}
	cluster.NodesCount = uint32(len(nodes.Items))

	version, err := cli.ServerVersion()
	if err != nil {
		logger.Error("get version failed:%s", err.Error())
	} else {
		cluster.Version = version.GitVersion
	}
	cluster.KubeClient = cli
	m.clusters = append(m.clusters, cluster)
	return cluster, nil
}

func (m *ClusterManager) Get(id string) (*types.Cluster, bool) {
	for _, c := range m.clusters {
		if c.GetID() == id {
			return c, true
		}
	}
	return nil, false
}

func (m *ClusterManager) List() []*types.Cluster {
	return m.clusters
}
