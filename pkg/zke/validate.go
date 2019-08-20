package zke

import (
	"fmt"
	"strings"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/set"
)

func validateConfigForCreate(c *types.Cluster) error {
	if err := validateClusterOptions(c); err != nil {
		return err
	}
	return validateNodes(c)
}

func validateConfigForUpdate(oldCluster, newCluster *types.Cluster) error {
	if err := validateNodes(newCluster); err != nil {
		return err
	}
	return validateUnallowDeleteNodes(oldCluster, newCluster)
}

func validateNodes(c *types.Cluster) error {
	if err := validateNodeRoleAndOption(c); err != nil {
		return err
	}
	if err := validateDuplicateNodes(c); err != nil {
		return err
	}
	if err := isSinlecloudInClusterNodes(c); err != nil {
		return err
	}
	return validateNodeCount(c)
}

func validateClusterOptions(c *types.Cluster) error {
	if len(c.Name) == 0 {
		return fmt.Errorf("cluster name can't empty")
	}
	if len(c.SSHUser) == 0 {
		return fmt.Errorf("cluster sshuser can't empty")
	}
	if len(c.SSHKey) == 0 {
		return fmt.Errorf("cluster sshkey can't empty")
	}
	if len(c.SingleCloudAddress) == 0 {
		return fmt.Errorf("singlecloud address can't empty")
	}
	if len(c.ClusterDomain) == 0 {
		return fmt.Errorf("cluster domain can't empty")
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

func validateNodeRoleAndOption(c *types.Cluster) error {
	for _, n := range c.Nodes {
		if len(n.Name) == 0 || len(n.Address) == 0 || len(n.Roles) == 0 {
			return fmt.Errorf("%s node name,address and roles cat't nil", n.Name)
		}
		if !n.HasRole(types.RoleControlPlane) && !n.HasRole(types.RoleWorker) {
			return fmt.Errorf("%s must be controlplane or worker", n.Name)
		}
		if n.HasRole(types.RoleControlPlane) && n.HasRole(types.RoleWorker) {
			return fmt.Errorf("%s controlplane node can't be worker", n.Name)
		}
	}
	return nil
}

func validateUnallowDeleteNodes(oldCluster, newCluster *types.Cluster) error {
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

func isSinlecloudInClusterNodes(c *types.Cluster) error {
	scIp := strings.Split(c.SingleCloudAddress, ":")[0]
	for _, n := range c.Nodes {
		if n.Address == scIp {
			return fmt.Errorf("singlecloud server %s cant't be an node", n.Address)
		}
	}
	return nil
}
