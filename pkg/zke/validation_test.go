package zke

import (
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func TestValidateDuplicateNodeName(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
	}

	n2 := types.Node{
		Name:    "master",
		Address: "192.168.1.2",
	}

	c := &types.Cluster{
		Nodes: []types.Node{n1, n2},
	}

	ut.NotEqual(t, validateDuplicateNodes(c), nil)
}

func TestValidateDuplicateNodeIp(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
	}

	n2 := types.Node{
		Name:    "worker",
		Address: "192.168.1.1",
	}

	c := &types.Cluster{
		Nodes: []types.Node{n1, n2},
	}

	ut.NotEqual(t, validateDuplicateNodes(c), nil)
}

func TestValidateNoControlplane(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
		Roles:   []types.NodeRole{types.RoleWorker, types.RoleEtcd},
	}

	n2 := types.Node{
		Name:    "worker",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleWorker, types.RoleEtcd},
	}

	c := &types.Cluster{
		Nodes: []types.Node{n1, n2},
	}

	ut.NotEqual(t, validateNodeCount(c), nil)
}

func TestValidateNoEtcd(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
		Roles:   []types.NodeRole{types.RoleControlPlane},
	}

	n2 := types.Node{
		Name:    "worker",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleWorker},
	}

	c := &types.Cluster{
		Nodes: []types.Node{n1, n2},
	}

	ut.NotEqual(t, validateNodeCount(c), nil)
}

func TestValidateNoWorker(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
		Roles:   []types.NodeRole{types.RoleControlPlane},
	}

	n2 := types.Node{
		Name:    "worker",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleControlPlane, types.RoleEtcd},
	}

	c := &types.Cluster{
		Nodes: []types.Node{n1, n2},
	}

	ut.NotEqual(t, validateNodeCount(c), nil)
}

func TestValidateNodeNameRolesAndAddress(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
		Roles:   []types.NodeRole{types.RoleControlPlane, types.RoleWorker, types.RoleEtcd},
	}

	n2 := types.Node{
		Name:    "worker",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleWorker, types.RoleWorker},
	}

	n3 := types.Node{
		Name:    "worker",
		Address: "xxxxx",
		Roles:   []types.NodeRole{types.RoleWorker},
	}

	n4 := types.Node{
		Name:    "Worker..1",
		Address: "xxxxx",
		Roles:   []types.NodeRole{types.RoleWorker},
	}

	c1 := &types.Cluster{
		Nodes: []types.Node{n1},
	}
	// test node role conflict case
	ut.NotEqual(t, validateNodeNameRoleAndAddress(c1), nil)
	// test node role duplicate case
	c2 := &types.Cluster{
		Nodes: []types.Node{n2},
	}
	ut.NotEqual(t, validateNodeNameRoleAndAddress(c2), nil)
	// test node address wrong
	c3 := &types.Cluster{
		Nodes: []types.Node{n3},
	}
	ut.NotEqual(t, validateNodeNameRoleAndAddress(c3), nil)
	// test node name not rfc1123subdomain
	c4 := &types.Cluster{
		Nodes: []types.Node{n4},
	}
	ut.NotEqual(t, validateNodeNameRoleAndAddress(c4), nil)
}

func TestValidateScAddress(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
		Roles:   []types.NodeRole{types.RoleControlPlane, types.RoleWorker, types.RoleEtcd},
	}

	n2 := types.Node{
		Name:    "worker",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleWorker},
	}

	c := &types.Cluster{
		Nodes:              []types.Node{n1, n2},
		SingleCloudAddress: "192.168.1.1:8088",
	}

	ut.NotEqual(t, validateScAddress(c), nil)
}

func TestValidateCannotDeleteNode(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
		Roles:   []types.NodeRole{types.RoleControlPlane, types.RoleWorker, types.RoleEtcd},
	}

	n2 := types.Node{
		Name:    "worker",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleWorker},
	}

	n3 := types.Node{
		Name:    "master2",
		Address: "192.168.1.3",
		Roles:   []types.NodeRole{types.RoleControlPlane, types.RoleEtcd},
	}

	c1 := &types.Cluster{
		Nodes: []types.Node{n1, n2},
	}

	c2 := &types.Cluster{
		Nodes: []types.Node{n2, n3},
	}

	ut.NotEqual(t, validateCannotDeleteNode(c1, c2), nil)
}

func TestValidateNodesRoleChanage(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
		Roles:   []types.NodeRole{types.RoleControlPlane, types.RoleWorker, types.RoleEtcd},
	}

	n2 := types.Node{
		Name:    "worker",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleWorker},
	}

	n3 := types.Node{
		Name:    "worker",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleWorker, types.RoleEdge},
	}

	c1 := &types.Cluster{
		Nodes: []types.Node{n1, n2},
	}

	c2 := &types.Cluster{
		Nodes: []types.Node{n1, n3},
	}

	ut.NotEqual(t, validateNodesRoleChanage(c1, c2), nil)
}

func TestValidateNodesNameChanage(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.1",
		Roles:   []types.NodeRole{types.RoleControlPlane, types.RoleWorker, types.RoleEtcd},
	}

	n2 := types.Node{
		Name:    "worker",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleWorker},
	}

	n3 := types.Node{
		Name:    "worker1",
		Address: "192.168.1.2",
		Roles:   []types.NodeRole{types.RoleWorker},
	}

	c1 := &types.Cluster{
		Nodes: []types.Node{n1, n2},
	}

	c2 := &types.Cluster{
		Nodes: []types.Node{n1, n3},
	}

	ut.NotEqual(t, validateNodesNameChanage(c1, c2), nil)
}

func TestValidateClusterCIDRAndIPs(t *testing.T) {
	// test clusterCidr wrong case
	c1 := &types.Cluster{
		ClusterCidr: "10.42.0.16",
		ServiceCidr: "10.43.0.0/16",
		ClusterUpstreamDNS: []string{
			"114.114.114.114",
			"223.5.5.5",
		},
		ClusterDNSServiceIP: "10.43.0.10",
	}
	ut.NotEqual(t, validateClusterCIDRAndIPs(c1), nil)
	// test serviceCidr wrong case
	c2 := &types.Cluster{
		ClusterCidr: "10.42.0.0/16",
		ServiceCidr: "10.43.0",
		ClusterUpstreamDNS: []string{
			"114.114.114.114",
			"223.5.5.5",
		},
		ClusterDNSServiceIP: "10.43.0.10",
	}
	ut.NotEqual(t, validateClusterCIDRAndIPs(c2), nil)
	// test clusterUpstreamDNS wrong case
	c3 := &types.Cluster{
		ClusterCidr: "10.42.0.0/16",
		ServiceCidr: "10.43.0.0/16",
		ClusterUpstreamDNS: []string{
			"114.114.114.1140",
			"223.5.5.5",
		},
		ClusterDNSServiceIP: "10.43.0.10",
	}
	ut.NotEqual(t, validateClusterCIDRAndIPs(c3), nil)
	// test clusterDnsServiceIP case
	c4 := &types.Cluster{
		ClusterCidr: "10.42.0.0/16",
		ServiceCidr: "10.43.0.0/16",
		ClusterUpstreamDNS: []string{
			"114.114.114.1140",
			"223.5.5.5",
		},
		ClusterDNSServiceIP: "10.43.0.1000",
	}
	ut.NotEqual(t, validateClusterCIDRAndIPs(c4), nil)
	c4.ClusterDNSServiceIP = "10.48.0.9"
	ut.NotEqual(t, validateClusterCIDRAndIPs(c4), nil)
}

func TestValidateClusterSSHKeyNotEmpty(t *testing.T) {
	c := &types.Cluster{
		SSHKey: "",
	}
	ut.NotEqual(t, validateClusterSSHKeyNotEmpty(c), nil)
}
