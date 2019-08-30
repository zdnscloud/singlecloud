package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/core/services"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/pkg/util"
	"github.com/zdnscloud/zke/types"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/zdnscloud/cement/errgroup"
)

const (
	etcdRoleLabel         = "node-role.kubernetes.io/etcd"
	controlplaneRoleLabel = "node-role.kubernetes.io/controlplane"
	workerRoleLabel       = "node-role.kubernetes.io/worker"
	StorageRoleLabel      = "node-role.kubernetes.io/storage"
	edgeRoleLabel         = "node-role.kubernetes.io/edge"
	cloudConfigFileName   = "/etc/kubernetes/cloud-config"
	authnWebhookFileName  = "/etc/kubernetes/kube-api-authn-webhook.yaml"
)

func (c *Cluster) TunnelHosts(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		c.InactiveHosts = make([]*hosts.Host, 0)
		uniqueHosts := hosts.GetUniqueHostList(c.EtcdHosts, c.ControlPlaneHosts, c.WorkerHosts, c.EdgeHosts)

		_, err := errgroup.Batch(uniqueHosts, func(h interface{}) (interface{}, error) {
			runHost := h.(*hosts.Host)
			if err := runHost.TunnelUp(ctx, c.DockerDialerFactory, c.Option.PrefixPath, c.Option.KubernetesVersion); err != nil {
				// Unsupported Docker version is NOT a connectivity problem that we can recover! So we bail out on it
				if strings.Contains(err.Error(), "Unsupported Docker version found") {
					return nil, err
				}
				return nil, fmt.Errorf("Failed to set up SSH tunneling for host [%s]: %s", runHost.Address, err.Error())
			}
			return nil, nil
		})
		if err != nil {
			return err
		}

		for _, host := range c.InactiveHosts {
			log.Warnf(ctx, "Removing host [%s] from node lists", host.Address)
			c.EtcdHosts = removeFromHosts(host, c.EtcdHosts)
			c.ControlPlaneHosts = removeFromHosts(host, c.ControlPlaneHosts)
			c.WorkerHosts = removeFromHosts(host, c.WorkerHosts)
			c.EdgeHosts = removeFromHosts(host, c.EdgeHosts)
			c.ZKEConfig.Nodes = removeFromZKENodes(host.ZKEConfigNode, c.ZKEConfig.Nodes)
		}
		return ValidateHostCount(c)
	}
}

func (c *Cluster) InvertIndexHosts() error {
	c.EtcdHosts = make([]*hosts.Host, 0)
	c.WorkerHosts = make([]*hosts.Host, 0)
	c.ControlPlaneHosts = make([]*hosts.Host, 0)
	c.EdgeHosts = make([]*hosts.Host, 0)
	for _, host := range c.Nodes {
		newHost := hosts.Host{
			ZKEConfigNode: host,
			ToAddLabels:   map[string]string{},
			ToDelLabels:   map[string]string{},
			ToAddTaints:   []string{},
			ToDelTaints:   []string{},
			DockerInfo: dockertypes.Info{
				DockerRootDir: "/var/lib/docker",
			},
		}
		for k, v := range host.Labels {
			newHost.ToAddLabels[k] = v
		}
		newHost.IgnoreDockerVersion = c.Option.IgnoreDockerVersion
		for _, role := range host.Role {
			log.Debugf("Host: " + host.Address + " has role: " + role)
			switch role {
			case services.ETCDRole:
				newHost.IsEtcd = true
				newHost.ToAddLabels[etcdRoleLabel] = "true"
				c.EtcdHosts = append(c.EtcdHosts, &newHost)
			case services.ControlRole:
				newHost.IsControl = true
				newHost.ToAddLabels[controlplaneRoleLabel] = "true"
				c.ControlPlaneHosts = append(c.ControlPlaneHosts, &newHost)
			case services.WorkerRole:
				newHost.IsWorker = true
				newHost.ToAddLabels[workerRoleLabel] = "true"
				c.WorkerHosts = append(c.WorkerHosts, &newHost)
			case services.EdgeRole:
				newHost.IsEdge = true
				newHost.ToAddLabels[edgeRoleLabel] = "true"
				c.EdgeHosts = append(c.EdgeHosts, &newHost)
			default:
				return fmt.Errorf("Failed to recognize host [%s] role %s", host.Address, role)
			}
		}
		if !newHost.IsEtcd {
			newHost.ToDelLabels[etcdRoleLabel] = "true"
		}
		if !newHost.IsControl {
			newHost.ToDelLabels[controlplaneRoleLabel] = "true"
		}
		if !newHost.IsWorker {
			newHost.ToDelLabels[workerRoleLabel] = "true"
		}
		if !newHost.IsStorage {
			newHost.ToDelLabels[StorageRoleLabel] = "true"
		}
		if !newHost.IsEdge {
			newHost.ToDelLabels[edgeRoleLabel] = "true"
		}
	}
	return nil
}

func (c *Cluster) SetUpHosts(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		if c.AuthnStrategies[AuthnX509Provider] {
			log.Infof(ctx, "[certificates] Deploying kubernetes certificates to Cluster nodes")
			forceDeploy := false

			hostList := hosts.GetUniqueHostList(c.EtcdHosts, c.ControlPlaneHosts, c.WorkerHosts, c.EdgeHosts)
			_, err := errgroup.Batch(hostList, func(h interface{}) (interface{}, error) {
				return nil, pki.DeployCertificatesOnPlaneHost(ctx, h.(*hosts.Host), c.ZKEConfig, c.Certificates, c.Image.CertDownloader, c.PrivateRegistriesMap, forceDeploy)
			})
			if err != nil {
				return err
			}

			if err := rebuildLocalAdminConfig(ctx, c); err != nil {
				return err
			}
			log.Infof(ctx, "[certificates] Successfully deployed kubernetes certificates to Cluster nodes")

			if c.Authentication.Webhook != nil {
				if err := deployFile(ctx, hostList, c.Image.Alpine, c.PrivateRegistriesMap, authnWebhookFileName, c.Authentication.Webhook.ConfigFile); err != nil {
					return err
				}
				log.Infof(ctx, "[%s] Successfully deployed authentication webhook config Cluster nodes", authnWebhookFileName)
			}
		}
		return nil
	}
}

func CheckEtcdHostsChanged(kubeCluster, currentCluster *Cluster) error {
	if currentCluster != nil {
		etcdChanged := hosts.IsHostListChanged(currentCluster.EtcdHosts, kubeCluster.EtcdHosts)
		if etcdChanged {
			return fmt.Errorf("Adding or removing Etcd nodes is not supported")
		}
	}
	return nil
}

func removeFromHosts(hostToRemove *hosts.Host, hostList []*hosts.Host) []*hosts.Host {
	for i := range hostList {
		if hostToRemove.Address == hostList[i].Address {
			return append(hostList[:i], hostList[i+1:]...)
		}
	}
	return hostList
}

func removeFromZKENodes(nodeToRemove types.ZKEConfigNode, nodeList []types.ZKEConfigNode) []types.ZKEConfigNode {
	for i := range nodeList {
		if nodeToRemove.Address == nodeList[i].Address {
			return append(nodeList[:i], nodeList[i+1:]...)
		}
	}
	return nodeList
}
