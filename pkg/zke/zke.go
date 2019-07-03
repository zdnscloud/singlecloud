package zke

import (
	"fmt"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	gok8sconfig "github.com/zdnscloud/gok8s/client/config"
	zkecmd "github.com/zdnscloud/zke/cmd"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	zketypes "github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

const (
	ClusterCreateFailed   = "CreateFailed"
	ClusterCreateComplete = "CreateComplete"
	ClusterCreateing      = "Createing"
	ClusterUpateing       = "updateing"
	ClusterUpateComplete  = "updated"
	ClusterUpateFailed    = "updateFailed"
)

type ZKEMsg struct {
	ClusterName   string
	ClusterState  *core.FullState
	ClusterConfig *zketypes.ZKEConfig
	KubeConfig    *rest.Config
	KubeClient    client.Client
	Status        string
	ErrorMsg      string
}

type ZKEEvent struct {
	State  *core.FullState
	Config *zketypes.ZKEConfig
}

func CreateCluster(eventCh chan ZKEEvent, msgCh chan ZKEMsg) {
	event := <-eventCh
	var msg = ZKEMsg{
		ClusterName:   event.Config.ClusterName,
		ClusterConfig: event.Config.DeepCopy(),
		Status:        ClusterCreateFailed,
	}
	defer func(msgCh chan ZKEMsg) {
		if r := recover(); r != nil {
			err := fmt.Errorf("zke err: %s", r)
			msg.ErrorMsg = err.Error()
			msgCh <- msg
		}
	}(msgCh)

	if err := zkecmd.ClusterRemoveFromRest(event.Config); err != nil {
		msg.ErrorMsg = err.Error()
		msgCh <- msg
	}

	state, err := zkecmd.ClusterUpFromRest(event.Config, &core.FullState{})
	if err != nil {
		msg.ErrorMsg = err.Error()
		msgCh <- msg
	}

	kubeConfigYaml := state.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	kubeConfig, err := gok8sconfig.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		msg.ErrorMsg = err.Error()
		msgCh <- msg
	}

	kubeClient, err := client.New(kubeConfig, client.Options{})
	if err != nil {
		msg.ErrorMsg = err.Error()
		msgCh <- msg
	}

	if err := deployZcloudProxy(kubeClient, event.Config.ClusterName, event.Config.SingleCloudAddress); err != nil {
		msg.ErrorMsg = err.Error()
		msgCh <- msg
	}

	msg.KubeClient = kubeClient
	msg.KubeConfig = kubeConfig
	msg.Status = ClusterCreateComplete
	msg.ClusterState = state
	msgCh <- msg
}

func UpdateCluster(eventCh chan ZKEEvent, msgCh chan ZKEMsg) {
	event := <-eventCh
	var msg = ZKEMsg{
		ClusterName:   event.Config.ClusterName,
		ClusterConfig: event.Config.DeepCopy(),
		Status:        ClusterUpateFailed,
	}
	defer func(msgCh chan ZKEMsg) {
		if r := recover(); r != nil {
			err := fmt.Errorf("zke err: %s", r)
			msg.ErrorMsg = err.Error()
			msgCh <- msg
		}
	}(msgCh)

	state, err := zkecmd.ClusterUpFromRest(event.Config, event.State)
	if err != nil {
		msg.ErrorMsg = err.Error()
		msgCh <- msg
	}

	kubeConfigYaml := state.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	kubeConfig, err := gok8sconfig.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		msg.ErrorMsg = err.Error()
		msgCh <- msg
	}

	kubeClient, err := client.New(kubeConfig, client.Options{})
	if err != nil {
		msg.ErrorMsg = err.Error()
		msgCh <- msg
	}

	msg.KubeClient = kubeClient
	msg.KubeConfig = kubeConfig
	msg.Status = ClusterUpateComplete
	msg.ClusterState = state
	msgCh <- msg
}

func ScClusterToZKEConfig(cluster *types.Cluster) *zketypes.ZKEConfig {
	config := &zketypes.ZKEConfig{
		ClusterName:        cluster.Name,
		SingleCloudAddress: cluster.SingleCloudAddress,
	}
	config.Option.SSHUser = cluster.SSHUser
	config.Option.SSHPort = cluster.SSHPort
	config.Option.SSHKey = cluster.SSHKey
	config.Option.ClusterCidr = cluster.ClusterCidr
	config.Option.ServiceCidr = cluster.ServiceCidr
	config.Option.ClusterDomain = cluster.ClusterDomain
	config.Option.ClusterDNSServiceIP = cluster.ClusterDNSServiceIP
	config.Option.ClusterUpstreamDNS = cluster.ClusterUpstreamDNS
	config.Network.Plugin = cluster.Network.Plugin

	config.Nodes = []zketypes.ZKEConfigNode{}

	for _, node := range cluster.Nodes {
		n := zketypes.ZKEConfigNode{
			NodeName: node.NodeName,
			Address:  node.Address,
			Role:     node.Role,
		}
		config.Nodes = append(config.Nodes, n)
	}

	if cluster.PrivateRegistrys != nil {
		config.PrivateRegistries = []zketypes.PrivateRegistry{}
		for _, pr := range cluster.PrivateRegistrys {
			npr := zketypes.PrivateRegistry{
				User:     pr.User,
				Password: pr.Password,
				URL:      pr.URL,
				CAcert:   pr.CAcert,
			}
			config.PrivateRegistries = append(config.PrivateRegistries, npr)
		}
	}
	return config
}

func GetNewConfigForAddNode(currentConfig *zketypes.ZKEConfig, node *types.Node) (*zketypes.ZKEConfig, error) {
	for _, n := range currentConfig.Nodes {
		if n.NodeName == node.Name || n.Address == node.Address {
			return currentConfig, fmt.Errorf("duplicate node")
		}
	}

	newConfig := currentConfig.DeepCopy()

	zkeNode := zketypes.ZKEConfigNode{
		NodeName: node.Name,
		Address:  node.Address,
		Role:     node.Roles,
	}

	newConfig.Nodes = append(newConfig.Nodes, zkeNode)
	return newConfig, nil
}

func GetNewConfigForDeleteNode(currentConfig *zketypes.ZKEConfig, nodeName string) (*zketypes.ZKEConfig, bool) {
	newConfig := currentConfig.DeepCopy()
	isExist := false

	for i, n := range newConfig.Nodes {
		if n.NodeName == nodeName {
			isExist = true
			newConfig.Nodes = append(newConfig.Nodes[:i], newConfig.Nodes[i+1:]...)
		}
	}

	return newConfig, isExist
}
