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

func TestValidateNodeRolesConflict(t *testing.T) {
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
		Nodes: []types.Node{n1, n2},
	}

	ut.NotEqual(t, validateNodeRoleConflict(c), nil)
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
