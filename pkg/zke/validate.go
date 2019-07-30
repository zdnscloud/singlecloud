package zke

import (
	"fmt"

	"github.com/zdnscloud/singlecloud/pkg/types"
)

func validateConfig(c *types.Cluster) error {
	if err := validateClusterOptions(c); err != nil {
		return err
	}
	if err := validateDuplicateNodes(c); err != nil {
		return err
	}
	if err := validateNodeOptions(c); err != nil {
		return err
	}
	if err := validateNodeCount(c); err != nil {
		return err
	}
	if err := validateNodeRole(c); err != nil {
		return err
	}
	return nil
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

func validateNodeOptions(c *types.Cluster) error {
	for _, n := range c.Nodes {
		if len(n.Name) == 0 || len(n.Address) == 0 || len(n.Roles) == 0 {
			return fmt.Errorf("node name,address and roles cat't nil")
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

func validateNodeRole(c *types.Cluster) error {
	for _, n := range c.Nodes {
		if !n.HasRole(types.RoleControlPlane) && !n.HasRole(types.RoleWorker) {
			return fmt.Errorf("%s must be controlplane or worker", n.Name)
		}
		if n.HasRole(types.RoleControlPlane) && n.HasRole(types.RoleWorker) {
			return fmt.Errorf("%s controlplane node can't be worker", n.Name)
		}
	}
	return nil
}
