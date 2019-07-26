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
	"k8s.io/client-go/rest"
)

func upCluster(ctx context.Context, config *zketypes.ZKEConfig, state *core.FullState, logger log.Logger) (*core.FullState, *rest.Config, client.Client, error) {
	newState, err := zkecmd.ClusterUpFromSingleCloud(ctx, config, state, logger)
	if err != nil {
		return newState, nil, nil, err
	}

	kubeConfigYaml := newState.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config
	k8sConfig, err := gok8sconfig.BuildConfig([]byte(kubeConfigYaml))
	if err != nil {
		return newState, k8sConfig, nil, err
	}

	kubeClient, err := client.New(k8sConfig, client.Options{})
	if err != nil {
		return newState, k8sConfig, kubeClient, err
	}

	if err := deployZcloudProxy(kubeClient, config.ClusterName, config.SingleCloudAddress); err != nil {
		return newState, k8sConfig, kubeClient, err
	}

	return newState, k8sConfig, kubeClient, nil
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

	if cluster.PrivateRegistries != nil {
		config.PrivateRegistries = []zketypes.PrivateRegistry{}
		for _, pr := range cluster.PrivateRegistries {
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

func ZKEClusterToSCCluster(zc *Cluster) *types.Cluster {
	sc := &types.Cluster{}
	sc.Name = zc.Name
	sc.SSHUser = zc.config.Option.SSHUser
	sc.SSHPort = zc.config.Option.SSHPort
	sc.SSHKey = zc.config.Option.SSHKey
	sc.ClusterCidr = zc.config.Option.ClusterCidr
	sc.ServiceCidr = zc.config.Option.ServiceCidr
	sc.ClusterDomain = zc.config.Option.ClusterDomain
	sc.ClusterDNSServiceIP = zc.config.Option.ClusterDNSServiceIP
	sc.ClusterUpstreamDNS = zc.config.Option.ClusterUpstreamDNS
	sc.SingleCloudAddress = zc.config.SingleCloudAddress

	sc.Network = types.ClusterNetwork{
		Plugin: zc.config.Network.Plugin,
	}

	sc.Nodes = []types.ClusterConfigNode{}
	for _, node := range zc.config.Nodes {
		n := types.ClusterConfigNode{
			NodeName: node.NodeName,
			Address:  node.Address,
			Role:     node.Role,
		}
		sc.Nodes = append(sc.Nodes, n)
	}

	if zc.config.PrivateRegistries != nil {
		sc.PrivateRegistries = []types.PrivateRegistry{}
		for _, pr := range zc.config.PrivateRegistries {
			npr := types.PrivateRegistry{
				User:     pr.User,
				Password: pr.Password,
				URL:      pr.URL,
				CAcert:   pr.CAcert,
			}
			sc.PrivateRegistries = append(sc.PrivateRegistries, npr)
		}
	}

	sc.SetID(zc.Name)
	sc.SetType(types.ClusterType)
	sc.SetCreationTimestamp(zc.CreateTime)
	sc.Status = zc.getStatus()
	return sc
}
