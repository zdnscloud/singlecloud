package domaintree

import (
	"github.com/zdnscloud/g53"
)

const RBT_MAX_LEVEL = g53.MAX_LABELS
const NORMAL_TREE_DEPTH = 5

type NodeChain struct {
	nodes          []*Node
	lastCompared   *Node
	lastComparison g53.NameComparisonResult
}

func NewNodeChain() *NodeChain {
	return &NodeChain{
		nodes: make([]*Node, 0, NORMAL_TREE_DEPTH),
	}
}

func (c *NodeChain) clear() {
	c.nodes = c.nodes[:0]
	c.lastCompared = nil
}

func (c *NodeChain) GetAbsoluteName() *g53.Name {
	if c.IsEmpty() {
		panic("get name on empty node chain")
	}

	nameCount := len(c.nodes)
	if nameCount == 1 {
		return c.nodes[0].name
	}

	names := [RBT_MAX_LEVEL]*g53.Name{}
	i := 0
	for j := nameCount - 1; j >= 0; j-- {
		names[i] = c.nodes[j].name
		i += 1
	}
	absoluteName, _ := names[0].Concat(names[1:nameCount]...)
	return absoluteName
}

func (c *NodeChain) IsEmpty() bool {
	return len(c.nodes) == 0
}

func (c *NodeChain) GetLevelCount() int {
	return len(c.nodes)
}

func (c *NodeChain) Top() *Node {
	if c.IsEmpty() {
		panic("top on empty chain")
	}
	return c.nodes[len(c.nodes)-1]
}

func (c *NodeChain) Pop() {
	if c.IsEmpty() {
		panic("pop on empty chain")
	}
	c.nodes = c.nodes[:(len(c.nodes) - 1)]
}

func (c *NodeChain) push(node *Node) {
	if len(c.nodes) == RBT_MAX_LEVEL {
		panic("too deep tree")
	}
	c.nodes = append(c.nodes, node)
}

func (c *NodeChain) LastComparison() g53.NameComparisonResult {
	return c.lastComparison
}
