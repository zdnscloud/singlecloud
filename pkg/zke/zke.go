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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

const (
	zcloudProxyReplicas   = 1
	zcloudProxyDeployName = "zcloud-proxy"
	zcloudProxyImage      = "zdnscloud/zcloud-proxy:v1.0.1"
	zcloudProxyNamespace  = "zcloud"
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

	if err := createOrUpdateZcloudProxy(kubeClient, config.ClusterName, config.SingleCloudAddress); err != nil {
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
	return newConfig
}

func createOrUpdateZcloudProxy(cli client.Client, clusterName, scAddress string) error {
	deploy := appsv1.Deployment{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{zcloudProxyNamespace, zcloudProxyDeployName}, &deploy)
	if apierrors.IsNotFound(err) {
		return cli.Create(context.TODO(), genZcloudProxyDeploy(clusterName, scAddress))
	}
	return cli.Update(context.TODO(), genZcloudProxyDeploy(clusterName, scAddress))
}

func genZcloudProxyDeploy(clusterName, scAddress string) *appsv1.Deployment {
	replicas := int32(zcloudProxyReplicas)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zcloudProxyDeployName,
			Namespace: zcloudProxyNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": zcloudProxyDeployName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": zcloudProxyDeployName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:    zcloudProxyDeployName,
							Image:   zcloudProxyImage,
							Command: []string{"agent"},
							Args:    []string{"-server", scAddress, "-cluster", clusterName},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
					DNSPolicy:     corev1.DNSClusterFirst,
				},
			},
		},
	}
}
