package zke

import (
	"net"
	"strconv"
	"strings"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/set"
	"github.com/zdnscloud/g53"
)

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

func isNodeRoleDuplicate(node types.Node) bool {
	roles := set.NewStringSet()
	for _, r := range node.Roles {
		roles.Add(string(r))
	}
	return len(roles) != len(node.Roles)
}

func isIPv4(input string) bool {
	ip := net.ParseIP(input)
	return ip != nil && ip.To4() != nil
}

func isCIDRv4(input string) bool {
	ip, _, err := net.ParseCIDR(input)
	return err == nil && ip.To4() != nil
}

func isCIDRv4Contains(network, ip string) bool {
	ipv4 := net.ParseIP(ip)
	if ipv4 == nil || ipv4.To4() == nil {
		return false
	}

	_, networkv4, err := net.ParseCIDR(network)
	return err == nil && networkv4.Contains(ipv4)
}

func isIPv4Host(input string) bool {
	contents := strings.Split(input, ":")
	if len(contents) != 2 {
		return false
	}

	ip := net.ParseIP(contents[0])
	port, err := strconv.Atoi(contents[1])
	if err != nil {
		return false
	}
	return ip != nil && ip.To4() != nil && port > 0 && port <= 65535
}

func getToDeleteNodes(old, new *types.Cluster) []string {
	newNodes := set.NewStringSet()
	toDeleteNodes := []string{}

	for _, n := range new.Nodes {
		newNodes.Add(n.Name)
	}

	for _, n := range old.Nodes {
		if !newNodes.Member(n.Name) {
			toDeleteNodes = append(toDeleteNodes, n.Name)
		}
	}
	return toDeleteNodes
}

func isDomainName(input string) bool {
	if _, err := g53.NameFromString(input); err != nil {
		return false
	}
	return true
}
