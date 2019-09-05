package zke

import (
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func TestValidateDuplicateNodeName(t *testing.T) {
	n1 := types.Node{
		Name:    "master",
		Address: "192.168.1.3",
	}

	n2 = types.Node{
		Name:    "master",
		Address: "192."

	c := &types.Cluster{
		Nodes: []types.Node{n1, n2, n},
	}

	ut.NotEqual(t, validateDuplicateNodes(c), nil)
}

func TestValidateDuplicateNodeIp(t *testing.T) {
	n := types.Node{
		Name:    "worker2",
		Address: n1.Address,
		Roles:   n2.Roles,
	}

	c := &types.Cluster{
		Nodes: []types.Node{n1, n2, n},
	}

	ut.NotEqual(t, validateDuplicateNodes(c), nil)
}

func TestValidateNoControlplane(t *testing.T) {
	n := types.Node{
		Name:    "worker2",
		Address: "192.168.1.3",
		Roles:   n2.Roles,
	}

	c := &types.Cluster{
		Nodes: []types.Node{n2, n},
	}

	ut.NotEqual(t, validateNodeCount(c), nil)
}

func TestValidateNoEtcd(t *testing.T) {
	n1 := types.Node{
		Name:    "worker2",
		Address: "192.168.1.3",
		Roles:   []types.NodeRole{types.RoleWorker},
	}

	n2 :=


	c := &types.Cluster{
		Nodes: []types.Node{n2, n},
	}

	ut.NotEqual(t, validateNodeCount(c), nil)
}

