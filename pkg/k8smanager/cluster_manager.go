package k8smanager

import (
	"fmt"
	"net/http"
	"time"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gok8s/exec"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/gorilla/websocket"
)

const (
	ZCloudNamespace = "zcloud"
	ZCloudAdmin     = "zcloud-cluster-admin"
	ZCloudReadonly  = "zcloud-cluster-readonly"

	OpenConsole = "console"
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

	executor, err := exec.New(k8sconf)
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
	cluster.Executor = executor
	m.clusters = append(m.clusters, cluster)

	initCluster(cluster)
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

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
}

func (m *ClusterManager) OpenConsole(id string, r *http.Request, w http.ResponseWriter) {
	cluster, found := m.Get(id)
	if found == false {
		logger.Warn("cluster %s isn't found to open console", id)
		return
	}

	conn_, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to set websocket upgrade: %+v", err)
		return
	}
	conn := newConnAdaptor(conn_)
	cmd := exec.Cmd{
		Path:   "/bin/sh",
		Stdin:  conn,
		Stdout: conn,
		Stderr: conn,
	}

	pod := exec.Pod{
		Namespace:          ZCloudNamespace,
		Name:               fmt.Sprintf("kubectl-console-%s", time.Now().String()),
		Image:              "lachlanevenson/k8s-kubectl:v1.13.3",
		ServiceAccountName: ZCloudReadonly,
	}

	if err := cluster.Executor.RunCmd(pod, cmd, 30*time.Second); err != nil {
		logger.Error("open console failed:%s", err.Error())
	}
}

func (m *ClusterManager) List() []*types.Cluster {
	return m.clusters
}

func initCluster(cluster *types.Cluster) error {
	cli := cluster.KubeClient
	if err := createNamespace(cli, ZCloudNamespace); err != nil {
		logger.Error("create namespace %s failed: %s", ZCloudNamespace, err.Error())
		return err
	}

	if err := createRole(cluster, ZCloudAdmin, ClusterAdmin); err != nil {
		return err
	}
	if err := createRole(cluster, ZCloudReadonly, ClusterAdmin); err != nil {
		return err
	}
	return nil
}

func createRole(cluster *types.Cluster, roleName string, role ClusterRole) error {
	cli := cluster.KubeClient
	if err := createServiceAccount(cli, roleName, ZCloudNamespace); err != nil {
		logger.Error("create service account %s failed: %s", roleName, err.Error())
		return err
	}

	if err := createClusterRole(cli, roleName, role); err != nil {
		logger.Error("create cluster role %s failed: %s", roleName, err.Error())
		return err
	}

	if err := createRoleBinding(cli, roleName, roleName, ZCloudNamespace); err != nil {
		logger.Error("create clusterRoleBinding %s failed: %s", ZCloudAdmin, err.Error())
		return err
	}

	return nil
}
