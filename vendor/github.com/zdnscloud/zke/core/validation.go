package core

import (
	"fmt"
	"strings"

	"github.com/zdnscloud/zke/core/services"

	"k8s.io/apimachinery/pkg/util/validation"
)

func (c *Cluster) ValidateCluster() error {
	if err := validateDuplicateNodes(c); err != nil {
		return err
	}
	if err := validateHostsOptions(c); err != nil {
		return err
	}
	if err := validateAuthOptions(c); err != nil {
		return err
	}
	if err := validateNetworkOptions(c); err != nil {
		return err
	}
	if err := validateIngressOptions(c); err != nil {
		return err
	}
	return validateServicesOptions(c)
}

func validateAuthOptions(c *Cluster) error {
	for strategy, enabled := range c.AuthnStrategies {
		if !enabled {
			continue
		}
		strategy = strings.ToLower(strategy)
		if strategy != AuthnX509Provider && strategy != AuthnWebhookProvider {
			return fmt.Errorf("Authentication strategy [%s] is not supported", strategy)
		}
	}
	if !c.AuthnStrategies[AuthnX509Provider] {
		return fmt.Errorf("Authentication strategy must contain [%s]", AuthnX509Provider)
	}
	return nil
}

func validateNetworkOptions(c *Cluster) error {
	if c.Network.Plugin != NoNetworkPlugin && c.Network.Plugin != FlannelNetworkPlugin && c.Network.Plugin != CalicoNetworkPlugin {
		return fmt.Errorf("Network plugin [%s] is not supported", c.Network.Plugin)
	}
	return nil
}

func validateHostsOptions(c *Cluster) error {
	for i, host := range c.Nodes {
		if len(host.Address) == 0 {
			return fmt.Errorf("Address for host (%d) is not provided", i+1)
		}
		if len(host.User) == 0 {
			return fmt.Errorf("User for host (%d) is not provided", i+1)
		}
		if len(host.Role) == 0 {
			return fmt.Errorf("Role for host (%d) is not provided", i+1)
		}
		if errs := validation.IsDNS1123Subdomain(host.NodeName); len(errs) > 0 {
			return fmt.Errorf("Hostname_override [%s] for host (%d) is not valid: %v", host.NodeName, i+1, errs)
		}
		for _, role := range host.Role {
			if role != services.ETCDRole && role != services.ControlRole && role != services.WorkerRole && role != services.StorageRole && role != services.EdgeRole {
				return fmt.Errorf("Role [%s] for host (%d) is not recognized", role, i+1)
			}
		}
	}
	return nil
}

func validateServicesOptions(c *Cluster) error {
	servicesOptions := map[string]string{
		"etcd_image":                               c.Core.Etcd.Image,
		"kube_api_image":                           c.Core.KubeAPI.Image,
		"kube_api_service_cluster_ip_range":        c.Core.KubeAPI.ServiceClusterIPRange,
		"kube_controller_image":                    c.Core.KubeController.Image,
		"kube_controller_service_cluster_ip_range": c.Core.KubeController.ServiceClusterIPRange,
		"kube_controller_cluster_cidr":             c.Core.KubeController.ClusterCIDR,
		"scheduler_image":                          c.Core.Scheduler.Image,
		"kubelet_image":                            c.Core.Kubelet.Image,
		"kubelet_cluster_dns_service":              c.Core.Kubelet.ClusterDNSServer,
		"kubelet_cluster_domain":                   c.Core.Kubelet.ClusterDomain,
		"kubelet_infra_container_image":            c.Core.Kubelet.InfraContainerImage,
		"kubeproxy_image":                          c.Core.Kubeproxy.Image,
	}
	for optionName, OptionValue := range servicesOptions {
		if len(OptionValue) == 0 {
			return fmt.Errorf("%s can't be empty", strings.Join(strings.Split(optionName, "_"), " "))
		}
	}
	// Validate external etcd information
	if len(c.Core.Etcd.ExternalURLs) > 0 {
		if len(c.Core.Etcd.CACert) == 0 {
			return fmt.Errorf("External CA Certificate for etcd can't be empty")
		}
		if len(c.Core.Etcd.Cert) == 0 {
			return fmt.Errorf("External Client Certificate for etcd can't be empty")
		}
		if len(c.Core.Etcd.Key) == 0 {
			return fmt.Errorf("External Client Key for etcd can't be empty")
		}
		if len(c.Core.Etcd.Path) == 0 {
			return fmt.Errorf("External etcd path can't be empty")
		}
	}
	return nil
}

func validateIngressOptions(c *Cluster) error {
	// Should be changed when adding more ingress types
	if c.Network.Ingress.Provider != DefaultIngressController && c.Network.Ingress.Provider != "none" {
		return fmt.Errorf("Ingress controller %s is incorrect", c.Network.Ingress.Provider)
	}
	return nil
}

func ValidateHostCount(c *Cluster) error {
	if len(c.EtcdHosts) == 0 && len(c.Core.Etcd.ExternalURLs) == 0 {
		failedEtcdHosts := []string{}
		for _, host := range c.InactiveHosts {
			if host.IsEtcd {
				failedEtcdHosts = append(failedEtcdHosts, host.Address)
			}
			return fmt.Errorf("Cluster must have at least one etcd plane host: failed to connect to the following etcd host(s) %v", failedEtcdHosts)
		}
		return fmt.Errorf("Cluster must have at least one etcd plane host: please specify one or more etcd in cluster config")
	}
	if len(c.EtcdHosts) > 0 && len(c.Core.Etcd.ExternalURLs) > 0 {
		return fmt.Errorf("Cluster can't have both internal and external etcd")
	}
	return nil
}

func validateDuplicateNodes(c *Cluster) error {
	for i := range c.Nodes {
		for j := range c.Nodes {
			if i == j {
				continue
			}
			if c.Nodes[i].Address == c.Nodes[j].Address {
				return fmt.Errorf("Cluster can't have duplicate node: %s", c.Nodes[i].Address)
			}
			if c.Nodes[i].NodeName == c.Nodes[j].NodeName {
				return fmt.Errorf("Cluster can't have duplicate node: %s", c.Nodes[i].NodeName)
			}
		}
	}
	return nil
}
