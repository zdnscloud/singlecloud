package zke

import (
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

func TestIsNodeRolesChanage(t *testing.T) {
	old := types.Node{
		Roles: []types.NodeRole{types.RoleWorker},
	}
	new1 := types.Node{
		Roles: []types.NodeRole{types.RoleWorker, types.RoleEdge},
	}
	new2 := types.Node{
		Roles: []types.NodeRole{types.RoleWorker},
	}
	ut.Equal(t, isNodeRolesChanage(old, new1), true)
	ut.Equal(t, isNodeRolesChanage(old, new2), false)
}

func TestIsNodeRoleDuplicate(t *testing.T) {
	n1 := types.Node{
		Roles: []types.NodeRole{types.RoleControlPlane, types.RoleEtcd},
	}
	n2 := types.Node{
		Roles: []types.NodeRole{types.RoleControlPlane, types.RoleControlPlane},
	}
	ut.Equal(t, isNodeRoleDuplicate(n1), false)
	ut.Equal(t, isNodeRoleDuplicate(n2), true)
}

func TestIsIPv4(t *testing.T) {
	ip1 := "1.1.1.1"
	ip2 := "1.1.1.267"
	ip3 := "xxxxx"
	ut.Equal(t, isIPv4(ip1), true)
	ut.Equal(t, isIPv4(ip2), false)
	ut.Equal(t, isIPv4(ip3), false)
}

func TestIsCIDRv4(t *testing.T) {
	c1 := "1.1.1.0/24"
	c2 := "1.1.1.3"
	c3 := "xxxxx"
	ut.Equal(t, isCIDRv4(c1), true)
	ut.Equal(t, isCIDRv4(c2), false)
	ut.Equal(t, isCIDRv4(c3), false)
}

func TestIsCIDRv4Contains(t *testing.T) {
	n := "1.1.1.0/24"
	ip1 := "1.1.1.9"
	ip2 := "2.2.2.2"
	ip3 := "xxxxx"
	ut.Equal(t, isCIDRv4Contains(n, ip1), true)
	ut.Equal(t, isCIDRv4Contains(n, ip2), false)
	ut.Equal(t, isCIDRv4Contains(n, ip3), false)
}

func TestIsIPv4Host(t *testing.T) {
	h1 := "1.1.1.1:8000"
	h2 := "1.1.1.1:90000"
	h3 := "xxxxx"
	ut.Equal(t, isIPv4Host(h1), true)
	ut.Equal(t, isIPv4Host(h2), false)
	ut.Equal(t, isIPv4Host(h3), false)
}
