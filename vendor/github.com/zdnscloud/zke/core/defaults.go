package core

import (
	"context"
	"strings"

	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/types"
)

const (
	DefaultSSHUser        = "zcloud"
	DefaultSSHPort        = "22"
	DefaultSSHKeyPath     = "~/.ssh/id_rsa"
	DefaultDockerSockPath = "/var/run/docker.sock"

	DefaultClusterName           = "local"
	DefaultK8sVersion            = types.DefaultK8s
	DefaultServiceClusterIPRange = "10.43.0.0/16"
	DefaultClusterCIDR           = "10.42.0.0/16"
	DefaultNodePortRange         = "30000-32767"
	DefaultClusterDomain         = "cluster.local"
	DefaultClusterDNSService     = "10.43.0.10"

	DefaultNetworkPlugin      = "flannel"
	DefaultFlannelBackendType = "vxlan"

	DefaultDNSProvider       = "coredns"
	DefaultAuthStrategy      = "x509"
	DefaultAuthorizationMode = "rbac"

	DefaultIngressController             = "nginx"
	DefaultEtcdBackupCreationPeriod      = "12h"
	DefaultEtcdBackupRetentionPeriod     = "72h"
	DefaultEtcdSnapshot                  = true
	DefaultEtcdBackupConfigIntervalHours = 12
	DefaultEtcdBackupConfigRetention     = 6

	DefaultEtcdHeartbeatIntervalName  = "heartbeat-interval"
	DefaultEtcdHeartbeatIntervalValue = "500"
	DefaultEtcdElectionTimeoutName    = "election-timeout"
	DefaultEtcdElectionTimeoutValue   = "5000"

	DefaultMonitorMetricsProvider = "metrics-server"

	DefaultIngressNodeSelector = "node-role.kubernetes.io/edge"

	DefaultAuthnWebhookFile = `
	apiVersion: v1
	kind: Config
	clusters:
	- name: Default
	  cluster:
		insecure-skip-tls-verify: true
		server: http://127.0.0.1:6440/v1/authenticate
	users:
	- name: Default
	  user:
		insecure-skip-tls-verify: true
	current-context: webhook
	contexts:
	- name: webhook
	  context:
		user: Default
		cluster: Default
	`
	DefaultAuthnCacheTimeout = "5s"
	DefaultConfigVersion     = "v1.0.10"
)

var DefaultUpstreamDNS = []string{"114.114.114.114", "223.5.5.5"}

func (c *Cluster) setClusterDefaults(ctx context.Context) error {
	if len(c.ClusterName) == 0 {
		c.ClusterName = DefaultClusterName
	}
	if len(c.ConfigVersion) == 0 {
		c.ConfigVersion = DefaultConfigVersion
	}

	c.setClusterOptionDefaults()
	c.setClusterImageDefaults()
	c.setClusterNodesDefaults()
	c.setClusterServicesDefaults()
	c.setClusterNetworkDefaults()
	c.setClusterSecurity()
	c.setPrivateRegistries()
	c.setClusterMonitor()
	return nil
}

func (c *Cluster) setClusterSecurity() {
	if len(c.Authorization.Mode) == 0 {
		c.Authorization.Mode = DefaultAuthorizationMode
	}
	c.setClusterAuthnAndAuthzDefaults()
}

func (c *Cluster) setPrivateRegistries() {
	if c.PrivateRegistries == nil {
		c.PrivateRegistries = []types.PrivateRegistry{}
	}
	for _, pr := range c.PrivateRegistries {
		if pr.URL == "" {
			pr.URL = docker.DockerRegistryURL
		}
		c.PrivateRegistriesMap[pr.URL] = pr
	}
}

func (c *Cluster) setClusterMonitor() {
	if len(c.Monitor.MetricsProvider) == 0 {
		c.Monitor.MetricsProvider = DefaultMonitorMetricsProvider
	}
	if c.Monitor.MetricsOptions == nil {
		c.Monitor.MetricsOptions = map[string]string{}
	}
}

func (c *Cluster) setClusterImageDefaults() {
	c.Image = types.AllK8sVersions[c.Option.KubernetesVersion]
}

func (c *Cluster) setClusterNetworkDefaults() {

	if len(c.Network.Plugin) == 0 {
		c.Network.Plugin = DefaultNetworkPlugin
	}

	if len(c.Network.DNS.Provider) == 0 {
		c.Network.DNS.Provider = DefaultDNSProvider
	}
	if c.Network.DNS.UpstreamNameservers == nil {
		c.Network.DNS.UpstreamNameservers = []string{}
	}
	if len(c.Network.DNS.UpstreamNameservers) == 0 {
		c.Network.DNS.UpstreamNameservers = c.Option.ClusterUpstreamDNS
	}
	if c.Network.DNS.ReverseCIDRs == nil {
		c.Network.DNS.ReverseCIDRs = []string{}
	}
	if c.Network.DNS.NodeSelector == nil {
		c.Network.DNS.NodeSelector = map[string]string{}
	}

	if len(c.Network.Ingress.Provider) == 0 {
		c.Network.Ingress.Provider = DefaultIngressController
	}
	if c.Network.Ingress.Options == nil {
		c.Network.Ingress.Options = map[string]string{}
	}
	if c.Network.Ingress.ExtraArgs == nil {
		c.Network.Ingress.ExtraArgs = map[string]string{}
	}
	if c.Network.Ingress.NodeSelector == nil {
		c.Network.Ingress.NodeSelector = map[string]string{}
	}
	c.Network.Ingress.NodeSelector[DefaultIngressNodeSelector] = "true"

}

func (c *Cluster) setClusterOptionDefaults() {
	if len(c.Option.SSHUser) == 0 {
		c.Option.SSHUser = DefaultSSHUser
	}

	if len(c.Option.SSHPort) == 0 {
		c.Option.SSHPort = DefaultSSHPort
	}

	if len(c.Option.SSHKeyPath) == 0 {
		c.Option.SSHKeyPath = DefaultSSHKeyPath
	}

	if len(c.Option.DockerSocket) == 0 {
		c.Option.DockerSocket = DefaultDockerSockPath
	}

	if len(c.Option.KubernetesVersion) == 0 {
		c.Option.KubernetesVersion = DefaultK8sVersion
	}

	if len(c.Option.ClusterCidr) == 0 {
		c.Option.ClusterCidr = DefaultClusterCIDR
	}

	if len(c.Option.ServiceCidr) == 0 {
		c.Option.ServiceCidr = DefaultServiceClusterIPRange
	}

	if len(c.Option.ClusterDomain) == 0 {
		c.Option.ClusterDomain = DefaultClusterDomain
	}

	if len(c.Option.ClusterDNSServiceIP) == 0 {
		c.Option.ClusterDNSServiceIP = DefaultClusterDNSService
	}

	if c.Option.ClusterUpstreamDNS == nil {
		c.Option.ClusterUpstreamDNS = []string{}
	}

	if len(c.Option.ClusterUpstreamDNS) == 0 {
		c.Option.ClusterUpstreamDNS = DefaultUpstreamDNS
	}

	if len(c.Option.PrefixPath) == 0 {
		c.Option.PrefixPath = "/"
	}
}

func (c *Cluster) setClusterNodesDefaults() {
	for i, host := range c.Nodes {
		if len(host.InternalAddress) == 0 {
			c.Nodes[i].InternalAddress = c.Nodes[i].Address
		}

		if len(host.NodeName) == 0 {
			// This is a temporary modification
			c.Nodes[i].NodeName = c.Nodes[i].Address
		}

		if len(host.User) == 0 {
			c.Nodes[i].User = c.Option.SSHUser
		}

		if len(host.SSHKey) == 0 {
			c.Nodes[i].SSHKey = c.Option.SSHKey
		}

		if len(host.SSHKeyPath) == 0 {
			c.Nodes[i].SSHKeyPath = c.Option.SSHKeyPath
		}

		if len(host.Port) == 0 {
			c.Nodes[i].Port = c.Option.SSHPort
		}

		if len(host.DockerSocket) == 0 {
			c.Nodes[i].DockerSocket = c.Option.DockerSocket
		}

		if host.Labels == nil {
			c.Nodes[i].Labels = map[string]string{}
		}

		c.Nodes[i].NodeName = strings.ToLower(c.Nodes[i].NodeName)
	}
}

func (c *Cluster) setClusterServicesDefaults() {
	c.setCoreEtcdDefaults()
	c.setCoreKubeApiDefaults()
	c.setCoreKubeControllerDefaults()
	c.setCoreSchedulerDefaults()
	c.setCoreKubeletDefaults()
	c.setCoreKubeproxyDefaults()
}

func (c *Cluster) setCoreEtcdDefaults() {
	c.Core.Etcd.Image = c.Image.Etcd

	// enable etcd snapshots by default
	if c.Core.Etcd.Snapshot == nil {
		defaultSnapshot := DefaultEtcdSnapshot
		c.Core.Etcd.Snapshot = &defaultSnapshot
	}
	if c.Core.Etcd.ExtraArgs == nil {
		c.Core.Etcd.ExtraArgs = make(map[string]string)
	}

	if len(c.Core.Etcd.Creation) == 0 {
		c.Core.Etcd.Creation = DefaultEtcdBackupCreationPeriod
	}
	if len(c.Core.Etcd.Retention) == 0 {
		c.Core.Etcd.Retention = DefaultEtcdBackupRetentionPeriod
	}
	if _, ok := c.Core.Etcd.ExtraArgs[DefaultEtcdElectionTimeoutName]; !ok {
		c.Core.Etcd.ExtraArgs[DefaultEtcdElectionTimeoutName] = DefaultEtcdElectionTimeoutValue
	}

	if _, ok := c.Core.Etcd.ExtraArgs[DefaultEtcdHeartbeatIntervalName]; !ok {
		c.Core.Etcd.ExtraArgs[DefaultEtcdHeartbeatIntervalName] = DefaultEtcdHeartbeatIntervalValue
	}

	if c.Core.Etcd.BackupConfig != nil {
		if c.Core.Etcd.BackupConfig.IntervalHours == 0 {
			c.Core.Etcd.BackupConfig.IntervalHours = DefaultEtcdBackupConfigIntervalHours
		}
		if c.Core.Etcd.BackupConfig.Retention == 0 {
			c.Core.Etcd.BackupConfig.Retention = DefaultEtcdBackupConfigRetention
		}
	}

	if c.Core.Etcd.ExtraArgs == nil {
		c.Core.Etcd.ExtraArgs = map[string]string{}
	}
	if c.Core.Etcd.ExtraBinds == nil {
		c.Core.Etcd.ExtraBinds = []string{}
	}
	if c.Core.Etcd.ExtraEnv == nil {
		c.Core.Etcd.ExtraEnv = []string{}
	}
	if c.Core.Etcd.ExternalURLs == nil {
		c.Core.Etcd.ExternalURLs = []string{}
	}
}

func (c *Cluster) setCoreKubeApiDefaults() {
	c.Core.KubeAPI.Image = c.Image.Kubernetes
	if len(c.Core.KubeAPI.ServiceClusterIPRange) == 0 {
		c.Core.KubeAPI.ServiceClusterIPRange = c.Option.ServiceCidr
	}
	if len(c.Core.KubeAPI.ServiceNodePortRange) == 0 {
		c.Core.KubeAPI.ServiceNodePortRange = DefaultNodePortRange
	}
	if c.Core.KubeAPI.ExtraArgs == nil {
		c.Core.KubeAPI.ExtraArgs = map[string]string{}
	}
	if c.Core.KubeAPI.ExtraBinds == nil {
		c.Core.KubeAPI.ExtraBinds = []string{}
	}
	if c.Core.KubeAPI.ExtraEnv == nil {
		c.Core.KubeAPI.ExtraEnv = []string{}
	}
}

func (c *Cluster) setCoreKubeControllerDefaults() {
	c.Core.KubeController.Image = c.Image.Kubernetes
	if len(c.Core.KubeController.ServiceClusterIPRange) == 0 {
		c.Core.KubeController.ServiceClusterIPRange = c.Option.ServiceCidr
	}
	if len(c.Core.KubeController.ClusterCIDR) == 0 {
		c.Core.KubeController.ClusterCIDR = c.Option.ClusterCidr
	}
	if c.Core.KubeController.ExtraArgs == nil {
		c.Core.KubeController.ExtraArgs = map[string]string{}
	}
	if c.Core.KubeController.ExtraBinds == nil {
		c.Core.KubeController.ExtraBinds = []string{}
	}
	if c.Core.KubeController.ExtraEnv == nil {
		c.Core.KubeController.ExtraEnv = []string{}
	}
}

func (c *Cluster) setCoreSchedulerDefaults() {
	c.Core.Scheduler.Image = c.Image.Kubernetes
	if c.Core.Scheduler.ExtraArgs == nil {
		c.Core.Scheduler.ExtraArgs = map[string]string{}
	}
	if c.Core.Scheduler.ExtraBinds == nil {
		c.Core.Scheduler.ExtraBinds = []string{}
	}
	if c.Core.Scheduler.ExtraEnv == nil {
		c.Core.Scheduler.ExtraEnv = []string{}
	}
}

func (c *Cluster) setCoreKubeletDefaults() {
	c.Core.Kubelet.Image = c.Image.Kubernetes

	if len(c.Core.Kubelet.ClusterDNSServer) == 0 {
		c.Core.Kubelet.ClusterDNSServer = c.Option.ClusterDNSServiceIP
	}
	if len(c.Core.Kubelet.ClusterDomain) == 0 {
		c.Core.Kubelet.ClusterDomain = c.Option.ClusterDomain
	}
	if len(c.Core.Kubelet.InfraContainerImage) == 0 {
		c.Core.Kubelet.InfraContainerImage = c.Image.PodInfraContainer
	}
	if c.Core.Kubelet.ExtraArgs == nil {
		c.Core.Kubelet.ExtraArgs = map[string]string{}
	}
	if c.Core.Kubelet.ExtraBinds == nil {
		c.Core.Kubelet.ExtraBinds = []string{}
	}
	if c.Core.Kubelet.ExtraEnv == nil {
		c.Core.Kubelet.ExtraEnv = []string{}
	}
}

func (c *Cluster) setCoreKubeproxyDefaults() {
	c.Core.Kubeproxy.Image = c.Image.Kubernetes
	if c.Core.Kubeproxy.ExtraArgs == nil {
		c.Core.Kubeproxy.ExtraArgs = map[string]string{}
	}
	if c.Core.Kubeproxy.ExtraBinds == nil {
		c.Core.Kubeproxy.ExtraBinds = []string{}
	}
	if c.Core.Kubeproxy.ExtraEnv == nil {
		c.Core.Kubeproxy.ExtraEnv = []string{}
	}
}

func (c *Cluster) setClusterAuthnAndAuthzDefaults() {
	setDefaultIfEmpty(&c.Authentication.Strategy, DefaultAuthStrategy)

	for _, strategy := range strings.Split(c.Authentication.Strategy, "|") {
		strategy = strings.ToLower(strings.TrimSpace(strategy))
		c.AuthnStrategies[strategy] = true
	}

	if c.AuthnStrategies[AuthnWebhookProvider] && c.Authentication.Webhook == nil {
		c.Authentication.Webhook = &types.AuthWebhookConfig{}
	}
	if c.Authentication.Webhook != nil {
		webhookConfigDefaultsMap := map[*string]string{
			&c.Authentication.Webhook.ConfigFile:   DefaultAuthnWebhookFile,
			&c.Authentication.Webhook.CacheTimeout: DefaultAuthnCacheTimeout,
		}
		for k, v := range webhookConfigDefaultsMap {
			setDefaultIfEmpty(k, v)
		}
	}

	if c.Authentication.SANs == nil {
		c.Authentication.SANs = []string{}
	}

	if c.Authorization.Options == nil {
		c.Authorization.Options = map[string]string{}
	}
}

func setDefaultIfEmpty(varName *string, defaultValue string) {
	if len(*varName) == 0 {
		*varName = defaultValue
	}
}
