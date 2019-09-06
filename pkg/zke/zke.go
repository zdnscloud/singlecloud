package zke

import (
	"context"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	gok8sconfig "github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gok8s/helper"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	zkecmd "github.com/zdnscloud/zke/cmd"
	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	zketypes "github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

func buildZKECluster(ctx context.Context, config *zketypes.ZKEConfig, state *core.FullState, logger log.Logger) (*core.FullState, *rest.Config, client.Client, error) {
	return upZKECluster(ctx, config, state, logger, true)
}

func updateZKECluster(ctx context.Context, config *zketypes.ZKEConfig, state *core.FullState, logger log.Logger) (*core.FullState, *rest.Config, client.Client, error) {
	return upZKECluster(ctx, config, state, logger, false)
}

func upZKECluster(ctx context.Context, config *zketypes.ZKEConfig, state *core.FullState, logger log.Logger, isNewCluster bool) (*core.FullState, *rest.Config, client.Client, error) {
	newState, err := zkecmd.ClusterUpFromSingleCloud(ctx, config, state, logger, isNewCluster)
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

	if err := deployZcloudProxy(kubeClient, config.ClusterName, config.SingleCloudAddress); err != nil {
		return newState, k8sConfig, kubeClient, err
	}

	return newState, k8sConfig, kubeClient, nil
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
	return config
}

func genZKEConfigForUpdateNodes(config *zketypes.ZKEConfig, sc *types.Cluster) *zketypes.ZKEConfig {
	newConfig := config.DeepCopy()
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
	return newConfig
}

func genZcloudProxyDeployYaml(clusterName string, scAddress string) string {
	return `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: zcloud-proxy
  namespace: zcloud
spec:
  replicas: 1
  selector:
    matchLabels:
      app: zcloud-proxy
  template:
    metadata:
      labels:
        app: zcloud-proxy
    spec:
      containers:
      - args:
        - -server
        - "` + scAddress + `"
        - -cluster
        - "` + clusterName + `"
        command:
        - agent
        image: zdnscloud/zcloud-proxy:v1.0.1
        imagePullPolicy: IfNotPresent
        name: zcloud-proxy
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      securityContext: {}`
}

func deployZcloudProxy(cli client.Client, clusterName, scAddress string) error {
	yaml := genZcloudProxyDeployYaml(clusterName, scAddress)
	return helper.CreateResourceFromYaml(cli, yaml)
}
