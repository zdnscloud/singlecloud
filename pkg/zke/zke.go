package zke

import (
	"encoding/json"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/singlecloud/pkg/types"
	zkecmd "github.com/zdnscloud/zke/cmd"
	zketypes "github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

const (
	CreateFailed  = "failed"
	CreateSuccess = "success"
)

type ZkeMsg struct {
	ClusterName  string
	KubeClient   client.Client
	KubeConfig   *rest.Config
	ClusterState string
	CreateStatus string
	ZKEConfig    string
	ErrorMsg     string
}

func CreateClusterUseZKE(clusterState string, clusterCh <-chan *types.Cluster, zkeMsgCh chan ZkeMsg) {
	cluster := <-clusterCh
	var msg = ZkeMsg{
		ClusterName:  cluster.Name,
		CreateStatus: CreateFailed,
	}
	zkeCluster := scClusterToZKECluster(cluster)

	err := zkecmd.ClusterRemoveFromRest(zkeCluster)
	if err != nil {
		msg.ErrorMsg = err.Error()
		zkeMsgCh <- msg
	}

	newClusterState, kubeConfig, err := zkecmd.ClusterUpFromRest(zkeCluster, clusterState)
	if err != nil {
		msg.ErrorMsg = err.Error()
		zkeMsgCh <- msg
	}

	k8sconf, err := config.BuildConfig([]byte(kubeConfig))
	if err != nil {
		msg.ErrorMsg = err.Error()
		zkeMsgCh <- msg
	}

	cli, err := client.New(k8sconf, client.Options{})
	if err != nil {
		msg.ErrorMsg = err.Error()
		zkeMsgCh <- msg
	}

	err = DeployZcloudProxy(cli, cluster.Name, cluster.SingleCloudAddress)
	if err != nil {
		msg.ErrorMsg = err.Error()
		zkeMsgCh <- msg
	}

	zkeConfigJson, err := json.Marshal(zkeCluster)
	if err != nil {
		msg.ErrorMsg = err.Error()
		zkeMsgCh <- msg
	}

	msg.KubeClient = cli
	msg.KubeConfig = k8sconf
	msg.CreateStatus = CreateSuccess
	msg.ClusterState = newClusterState
	msg.ZKEConfig = string(zkeConfigJson)
	zkeMsgCh <- msg
}

func UpdateClusterUseZKE(clusterState string, zkeConfigJson, action string, nodeCh <-chan *types.Node, zkeMsgCh chan ZkeMsg) {
	node := <-nodeCh
	var msg = ZkeMsg{}
	zkeCluster, err := updateZKEConfigForNode(zkeConfigJson, action, node)
	if err != nil {
		msg.ErrorMsg = err.Error()
	}
	msg.ErrorMsg = zkeCluster.ClusterName

	newClusterState, kubeConfig, err := zkecmd.ClusterUpFromRest(zkeCluster, clusterState)
	if err != nil {
		msg.ErrorMsg = err.Error()
		zkeMsgCh <- msg
	}

	k8sconf, err := config.BuildConfig([]byte(kubeConfig))
	if err != nil {
		msg.ErrorMsg = err.Error()
	}

	cli, err := client.New(k8sconf, client.Options{})
	if err != nil {
		msg.ErrorMsg = err.Error()
	}

	msg.KubeClient = cli
	msg.KubeConfig = k8sconf
	msg.CreateStatus = CreateSuccess
	msg.ClusterState = newClusterState
	msg.ZKEConfig = string(zkeConfigJson)
	zkeMsgCh <- msg
}

func scClusterToZKECluster(scCluster *types.Cluster) *zketypes.ZKEConfig {
	zkeCluster := &zketypes.ZKEConfig{
		ClusterName:    scCluster.Name,
		SingleCloudUrl: scCluster.SingleCloudAddress,
	}
	zkeCluster.Option.SSHUser = scCluster.Option.SSHUser
	zkeCluster.Option.SSHPort = scCluster.Option.SSHPort
	zkeCluster.Option.SSHKey = scCluster.Option.SSHKey
	zkeCluster.Option.ClusterCidr = scCluster.Option.ClusterCidr
	zkeCluster.Option.ServiceCidr = scCluster.Option.ServiceCidr
	zkeCluster.Option.ClusterDomain = scCluster.Option.ClusterDomain
	zkeCluster.Option.ClusterDNSServiceIP = scCluster.Option.ClusterDNSServiceIP
	zkeCluster.Option.ClusterUpstreamDNS = scCluster.Option.ClusterUpstreamDNS
	zkeCluster.Network.Plugin = scCluster.Network.Plugin

	zkeCluster.Nodes = []zketypes.ZKEConfigNode{}

	for _, node := range scCluster.Nodes {
		n := zketypes.ZKEConfigNode{
			NodeName: node.NodeName,
			Address:  node.Address,
			Role:     node.Role,
		}
		zkeCluster.Nodes = append(zkeCluster.Nodes, n)
	}

	if scCluster.PrivateRegistrys != nil {
		zkeCluster.PrivateRegistries = []zketypes.PrivateRegistry{}
		for _, pr := range scCluster.PrivateRegistrys {
			npr := zketypes.PrivateRegistry{
				User:     pr.User,
				Password: pr.Password,
				URL:      pr.URL,
				CAcert:   pr.CAcert,
			}
			zkeCluster.PrivateRegistries = append(zkeCluster.PrivateRegistries, npr)
		}
	}
	return zkeCluster
}

func updateZKEConfigForNode(zkeConfigJson string, action string, node *types.Node) (*zketypes.ZKEConfig, error) {
	zkeConfig := &zketypes.ZKEConfig{}

	err := json.Unmarshal([]byte(zkeConfigJson), zkeConfig)
	if err != nil {
		return zkeConfig, err
	}

	zkeNode := zketypes.ZKEConfigNode{
		NodeName: node.Name,
		Address:  node.Address,
		Role:     node.Roles,
	}

	switch action {
	case "create":
		zkeConfig.Nodes = append(zkeConfig.Nodes, zkeNode)
	case "delete":
		for i, n := range zkeConfig.Nodes {
			if n.NodeName == zkeNode.NodeName {
				zkeConfig.Nodes = append(zkeConfig.Nodes[:i], zkeConfig.Nodes[i+1:]...)
			}
		}
	}
	return zkeConfig, nil
}
