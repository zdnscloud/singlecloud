package zke

import (
	"fmt"
	"strings"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/set"
)

type createValidator func(c *types.Cluster) error
type updateValidator func(oldCluster, newCluster *types.Cluster) error

var createValidators = []createValidator{
	validateClusterCIDRAndIPs,
	validateDuplicateNodes,
	validateNodeCount,
	validateNodeNameRoleAndAddress,
	validateScAddress,
	validateLBConfig,
}

var updateValidators = []updateValidator{
	validateCannotDeleteNode,
	validateNodesRoleChanage,
	validateNodesNameChanage,
}

func validateConfigForCreate(c *types.Cluster) error {
	for _, f := range createValidators {
		if err := f(c); err != nil {
			return err
		}
	}
	return validateClusterSSHKeyNotEmpty(c)
}

func validateConfigForUpdate(oldCluster, newCluster *types.Cluster, nl NodeListener, currentCluster *Cluster) error {
	for _, f := range createValidators {
		if err := f(newCluster); err != nil {
			return err
		}
	}

	for _, f := range updateValidators {
		if err := f(oldCluster, newCluster); err != nil {
			return err
		}
	}
	return validateToDeleteStorageNodes(oldCluster, newCluster, nl, currentCluster)
}

func validateToDeleteStorageNodes(oldCluster, newCluster *types.Cluster, nl NodeListener, currentCluster *Cluster) error {
	if currentCluster.KubeClient == nil {
		return nil
	}
	toDeleteNodes := getToDeleteNodes(oldCluster, newCluster)
	for _, n := range toDeleteNodes {
		isStorage, err := nl.IsStorageNode(currentCluster, n)
		if err != nil {
			return fmt.Errorf("validateToDeleteStorageNodes err %s", err.Error())
		}
		if isStorage {
			return fmt.Errorf("node %s used by storage,please delete it from storage first", n)
		}
	}
	return nil
}

func validateLBConfig(c *types.Cluster) error {
	if !c.LoadBalance.Enable {
		return nil
	}
	if !isIPv4(c.LoadBalance.MasterServer) && !isIPv4Host(c.LoadBalance.MasterServer) {
		return fmt.Errorf("loadbalance master server must be an ipv4 address or an ipv4 host")
	}
	if c.LoadBalance.BackupServer != "" && !isIPv4(c.LoadBalance.BackupServer) && !isIPv4Host(c.LoadBalance.BackupServer) {
		return fmt.Errorf("loadbalance backup server must be an ipv4 address or an ipv4 host")
	}
	if c.LoadBalance.User == "" || c.LoadBalance.Password == "" {
		return fmt.Errorf("loadbalance user and password can't be empty")
	}
	return nil
}

func validateDuplicateNodes(c *types.Cluster) error {
	for i := range c.Nodes {
		for j := range c.Nodes {
			if i == j {
				continue
			}
			if c.Nodes[i].Address == c.Nodes[j].Address {
				return fmt.Errorf("duplicate node address %s", c.Nodes[i].Address)
			}
			if c.Nodes[i].Name == c.Nodes[j].Name {
				return fmt.Errorf("duplicate node name %s", c.Nodes[i].Name)
			}
		}
	}
	return nil
}

func validateNodeCount(c *types.Cluster) error {
	var hasControlplane bool
	var hasEtcd bool
	var hasWorker bool
	for _, n := range c.Nodes {
		if n.HasRole(types.RoleControlPlane) {
			hasControlplane = true
		}
		if n.HasRole(types.RoleEtcd) {
			hasEtcd = true
		}
		if n.HasRole(types.RoleWorker) {
			hasWorker = true
		}
	}
	if !hasControlplane || !hasEtcd || !hasWorker {
		return fmt.Errorf("a cluster must has at least one controlplane, one etcd and one worker node")
	}
	return nil
}

func validateNodeNameRoleAndAddress(c *types.Cluster) error {
	for _, n := range c.Nodes {
		if !n.HasRole(types.RoleControlPlane) && !n.HasRole(types.RoleWorker) {
			return fmt.Errorf("%s must be controlplane or worker", n.Name)
		}
		if n.HasRole(types.RoleControlPlane) && n.HasRole(types.RoleWorker) {
			return fmt.Errorf("%s controlplane node can't be worker", n.Name)
		}
		if isNodeRoleDuplicate(n) {
			return fmt.Errorf("%s node has duplicate role", n.Name)
		}
		if !isIPv4(n.Address) {
			return fmt.Errorf("%s node address isn't an ipv4 address", n.Address)
		}
	}
	return nil
}

func validateCannotDeleteNode(oldCluster, newCluster *types.Cluster) error {
	cpHosts := set.NewStringSet()
	etcdHosts := set.NewStringSet()

	for _, n := range newCluster.Nodes {
		if n.HasRole(types.RoleControlPlane) {
			cpHosts.Add(n.Address)
		}
		if n.HasRole(types.RoleEtcd) {
			etcdHosts.Add(n.Address)
		}
	}

	for _, n := range oldCluster.Nodes {
		if n.HasRole(types.RoleControlPlane) {
			if !cpHosts.Member(n.Address) {
				return fmt.Errorf("controlplane node only can add, %s not in new config", n.Address)
			}
		}
		if n.HasRole(types.RoleEtcd) {
			if !etcdHosts.Member(n.Address) {
				return fmt.Errorf("etcd node only can add, %s not in new config", n.Address)
			}
		}
	}
	return nil
}

func validateScAddress(c *types.Cluster) error {
	if !isIPv4Host(c.SingleCloudAddress) {
		return fmt.Errorf("singlecloud address must be an ipv4 host such as 10.10.10.10:8000")
	}
	scIp := strings.Split(c.SingleCloudAddress, ":")[0]
	for _, n := range c.Nodes {
		if n.Address == scIp {
			return fmt.Errorf("singlecloud server %s cant't be an node", n.Address)
		}
	}
	return nil
}

func validateNodesRoleChanage(oldCluster, newCluster *types.Cluster) error {
	for _, old := range oldCluster.Nodes {
		for _, new := range newCluster.Nodes {
			if old.Address == new.Address {
				if isNodeRolesChanage(old, new) {
					return fmt.Errorf("don't support chanage node roles [%s]", old.Address)
				}
			}
		}
	}
	return nil
}

func validateNodesNameChanage(oldCluster, newCluster *types.Cluster) error {
	for _, old := range oldCluster.Nodes {
		for _, new := range newCluster.Nodes {
			if old.Address == new.Address {
				if old.Name != new.Name {
					return fmt.Errorf("don't support chanage node name [%s]", old.Address)
				}
			}
		}
	}
	return nil
}

func validateClusterCIDRAndIPs(c *types.Cluster) error {
	if !isIPv4(c.ClusterDNSServiceIP) {
		return fmt.Errorf("clusterDnsServiceIP %s isn't an ipv4 address", c.ClusterDNSServiceIP)
	}
	for _, ns := range c.ClusterUpstreamDNS {
		if !isIPv4(ns) {
			return fmt.Errorf("clusterUpstreamDNS %s isn't an ipv4 address", ns)
		}
	}
	if !isCIDRv4(c.ClusterCidr) {
		return fmt.Errorf("clusterCidr isn't an ipv4 CIDR")
	}
	if !isCIDRv4(c.ServiceCidr) {
		return fmt.Errorf("serviceCidr isn't an ipv4 CIDR")
	}
	if !isCIDRv4Contains(c.ServiceCidr, c.ClusterDNSServiceIP) {
		return fmt.Errorf("clusterDnsServiceIP %s not in serviceCidr %s", c.ClusterDNSServiceIP, c.ServiceCidr)
	}
	return nil
}

func validateClusterSSHKeyNotEmpty(c *types.Cluster) error {
	if len(c.SSHKey) == 0 {
		return fmt.Errorf("cluster sshkey is empty")
	}
	return nil
}
