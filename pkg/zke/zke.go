package zke

import (
	"context"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	gok8sconfig "github.com/zdnscloud/gok8s/client/config"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	zkecmd "github.com/zdnscloud/zke/cmd"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	zketypes "github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

func upZKECluster(ctx context.Context, config *zketypes.ZKEConfig, state *core.FullState, logger log.Logger) (*core.FullState, *rest.Config, client.Client, error) {
	newState, err := zkecmd.ClusterUpFromSingleCloud(ctx, config, state, logger)
	if err != nil {
		return newState, nil, nil, err
	}

	kubeConfigYaml := newState.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConfig, err := gok8sconfig.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		return newState, k8sConfig, nil, err
	}

	var options client.Options
	options.Scheme = client.GetDefaultScheme()
	storagev1.AddToScheme(options.Scheme)
	kubeClient, err := client.New(k8sConfig, options)
	if err != nil {
		return newState, k8sConfig, kubeClient, err
	}

	return newState, k8sConfig, kubeClient, nil
}

func removeZKECluster(ctx context.Context, config *zketypes.ZKEConfig, logger log.Logger) error {
	return zkecmd.ClusterRemoveFromSingleCloud(ctx, config, logger)
}

func genZKEConfig(cluster *types.Cluster) *zketypes.ZKEConfig {
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
	config.Network.Iface = cluster.Network.Iface

	config.Nodes = []zketypes.ZKEConfigNode{}
	for _, node := range cluster.Nodes {
		n := zketypes.ZKEConfigNode{
			NodeName: node.Name,
			Address:  node.Address,
		}
		for _, role := range node.Roles {
			n.Role = append(n.Role, string(role))
		}
		config.Nodes = append(config.Nodes, n)
	}

	if cluster.LoadBalance.MasterServer != "" {
		config.LoadBalance.Enable = true
		config.LoadBalance.MasterServer = cluster.LoadBalance.MasterServer
		config.LoadBalance.BackupServer = cluster.LoadBalance.BackupServer
		config.LoadBalance.User = cluster.LoadBalance.User
		config.LoadBalance.Password = cluster.LoadBalance.Password
	}

	return config
}

func genZKEConfigForUpdate(config *zketypes.ZKEConfig, sc *types.Cluster) *zketypes.ZKEConfig {
	newConfig := config.DeepCopy()
	newConfig.Option.SSHUser = sc.SSHUser
	newConfig.Option.SSHPort = sc.SSHPort
	if sc.SSHKey != "" {
		newConfig.Option.SSHKey = sc.SSHKey
	}
	newConfig.SingleCloudAddress = sc.SingleCloudAddress
	newConfig.Nodes = []zketypes.ZKEConfigNode{}
	for _, node := range sc.Nodes {
		n := zketypes.ZKEConfigNode{
			NodeName: node.Name,
			Address:  node.Address,
		}
		for _, role := range node.Roles {
			n.Role = append(n.Role, string(role))
		}
		newConfig.Nodes = append(newConfig.Nodes, n)
	}

	newConfig.LoadBalance.MasterServer = sc.LoadBalance.MasterServer
	newConfig.LoadBalance.BackupServer = sc.LoadBalance.BackupServer
	newConfig.LoadBalance.User = sc.LoadBalance.User
	newConfig.LoadBalance.Password = sc.LoadBalance.Password
	return newConfig
}
