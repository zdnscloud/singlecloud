package netradix

import (
	"net"
)

type NetRadixTree struct {
	tree *Tree
}

func NewNetRadixTree() *NetRadixTree {
	return &NetRadixTree{
		tree: NewTree(0),
	}
}

func (rtree *NetRadixTree) Add(subnet string, udata interface{}) error {
	return rtree.tree.AddCIDR(subnet, udata)
}

func (rtree *NetRadixTree) Delete(subnet string) error {
	return rtree.tree.DeleteCIDR(subnet)
}

func (rtree *NetRadixTree) SearchBest(addr net.IP) (interface{}, bool) {
	ipUint32, isV4 := ipToUint32(addr)
	var node interface{}
	if isV4 {
		node = rtree.tree.Find32(ipUint32, 0xffffffff)
	} else {
		node = rtree.tree.find(addr, net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	}

	if node == nil {
		return nil, false
	} else {
		return node, true
	}
}

func ipToUint32(ip net.IP) (uint32, bool) {
	ipbytes := ip.To4()
	if ipbytes == nil {
		return 0, false
	}

	return uint32(ipbytes[0])<<24 | uint32(ipbytes[1])<<16 | uint32(ipbytes[2])<<8 | uint32(ipbytes[3]), true
}
