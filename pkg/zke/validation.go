package zke

import (
	"fmt"
	"net"
	"strings"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/set"
)

type createValidator func(c *types.Cluster) error
type updateValidator func(oldCluster, newCluster *types.Cluster) error

var createValidators = []createValidator{
	validateDuplicateNodes,
	validateNodeCount,
	validateNodeRoleConflict,
	validateScAddress,
}

var updateValidators = []updateValidator{
	validateCannotDeleteNode,
	validateNodesRoleChanage,
}

func validateConfigForCreate(c *types.Cluster) error {
	fmt.Println(c)
	for _, f := range createValidators {
		if err := f(c); err != nil {
			return err
		}
	}
	return nil
}

func validateConfigForUpdate(oldCluster, newCluster *types.Cluster) error {
	if err := validateConfigForCreate(newCluster); err != nil {
		return err
	}

	for _, f := range updateValidators {
		if err := f(oldCluster, newCluster); err != nil {
			return err
		}
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

func validateNodeRoleConflict(c *types.Cluster) error {
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

func isNodeRolesChanage(oldNode, newNode types.Node) bool {
	oldRoles := set.NewStringSet()
	newRoles := set.NewStringSet()

	for _, r := range oldNode.Roles {
		oldRoles.Add(string(r))
	}

	for _, r := range newNode.Roles {
		newRoles.Add(string(r))
	}

	return !newRoles.Equal(oldRoles)
}

func isIPv4(input string) bool {
	ip := net.ParseIP(input)
	return ip != nil && ip.To4() != nil
}

func isCIDRv4(input string) bool {
	ip, _, err := net.ParseCIDR(input)
	return err == nil && ip.To4() != nil
}

func isIPv4Belong(ip, network string) bool {
	ipv4 := net.ParseIP(ip)
	if ipv4 == nil || ipv4.To4() == nil {
		return false
	}

	_, networkv4, err := net.ParseCIDR(network)
	return err == nil && networkv4.Contains(ipv4)
}

func isCIDRConflict(cidr1, cidr2 string) bool {
	return false
}

func isIP4Addr(input string) bool {
	if idx := strings.LastIndex(input, ":"); idx != -1 {
		input = input[0:idx]
		port := input[idx]
	}

	ip := net.ParseIP(input)

	return ip != nil && ip.To4() != nil
}
