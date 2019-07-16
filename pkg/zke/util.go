package zke

import (
	"context"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	gok8sconfig "github.com/zdnscloud/gok8s/client/config"
	zkecmd "github.com/zdnscloud/zke/cmd"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	zketypes "github.com/zdnscloud/zke/types"
)

func createCluster(ctx context.Context, cluster *ZKECluster, eventCh chan Event) {
	var event = Event{
		ID:     cluster.Config.ClusterName,
		Status: types.CSCreateFailed,
	}
	defer func(eventCh chan Event) {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			eventCh <- event
		}
	}(eventCh)

	if err := zkecmd.ClusterRemoveFromRest(ctx, cluster.Config, cluster.logCh); err != nil {
		eventCh <- event
		log.Errorf("zke err info %s", err)
		return
	}

	state, err := zkecmd.ClusterUpFromRest(ctx, cluster.Config, &core.FullState{}, cluster.logCh)
	if err != nil {
		eventCh <- event
		log.Errorf("zke err info %s", err)
		return
	}

	kubeConfigYaml := state.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConfig, err := gok8sconfig.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		eventCh <- event
		log.Errorf("zke err info %s", err)
		return
	}

	kubeClient, err := client.New(k8sConfig, client.Options{})
	if err != nil {
		eventCh <- event
		log.Errorf("zke err info %s", err)
		return
	}

	if err := deployZcloudProxy(kubeClient, cluster.Config.ClusterName, cluster.Config.SingleCloudAddress); err != nil {
		eventCh <- event
		log.Errorf("zke err info %s", err)
		return
	}

	event.KubeClient = kubeClient
	event.K8sConfig = k8sConfig
	event.Status = types.CSCreateSuccess
	event.State = state
	eventCh <- event
	return
}

func updateCluster(ctx context.Context, cluster *ZKECluster, eventCh chan Event) {
	var event = Event{
		ID:     cluster.Config.ClusterName,
		Status: types.CSUpdateFailed,
	}
	defer func(eventCh chan Event) {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			eventCh <- event
		}
	}(eventCh)

	state, err := zkecmd.ClusterUpFromRest(ctx, cluster.Config, cluster.State, cluster.logCh)
	if err != nil {
		eventCh <- event
		log.Errorf("zke err info %s", err)
		return
	}

	kubeConfigYaml := state.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConfig, err := gok8sconfig.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		eventCh <- event
		log.Errorf("zke err info %s", err)
		return
	}

	kubeClient, err := client.New(k8sConfig, client.Options{})
	if err != nil {
		eventCh <- event
		log.Errorf("zke err info %s", err)
		return
	}

	event.KubeClient = kubeClient
	event.K8sConfig = k8sConfig
	event.Status = types.CSUpdateSuccess
	event.State = state
	eventCh <- event
	return
}

func scClusterToZKEConfig(cluster *types.Cluster) (*zketypes.ZKEConfig, error) {
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

	if err := validateConfig(config); err != nil {
		return config, err
	}

	return config, nil
}

func getNewConfigForAddNode(currentConfig *zketypes.ZKEConfig, node *types.Node) (*zketypes.ZKEConfig, error) {
	newConfig := currentConfig.DeepCopy()

	zkeNode := zketypes.ZKEConfigNode{
		NodeName: node.Name,
		Address:  node.Address,
		Role:     node.Roles,
	}

	newConfig.Nodes = append(newConfig.Nodes, zkeNode)

	if err := validateConfig(newConfig); err != nil {
		return currentConfig, err
	}

	return newConfig, nil
}

func getNewConfigForDeleteNode(currentConfig *zketypes.ZKEConfig, nodeName string) (*zketypes.ZKEConfig, error) {
	newConfig := currentConfig.DeepCopy()

	for i, n := range newConfig.Nodes {
		if n.NodeName == nodeName {
			newConfig.Nodes = append(newConfig.Nodes[:i], newConfig.Nodes[i+1:]...)
		}
	}

	if err := validateConfig(newConfig); err != nil {
		return currentConfig, err
	}

	return newConfig, nil
}
