package domaintree

import (
	"github.com/zdnscloud/g53"
)

type RBNodeFlag uint32

const (
	NF_CALLBACK RBNodeFlag = 1
	NF_USER1    RBNodeFlag = 0x80000000
)

type RBNodeColor int

const (
	BLACK RBNodeColor = 0
	RED   RBNodeColor = 1
)

func (color RBNodeColor) String() string {
	if color == BLACK {
		return "black"
	} else {
		return "red"
	}
}

type Node struct {
	parent *Node
	left   *Node
	right  *Node
	color  RBNodeColor

	down *Node
	flag RBNodeFlag
	name *g53.Name
	data interface{}
}

var NULL_NODE *Node

func init() {
	if NULL_NODE == nil {
		NULL_NODE = &Node{
			color: BLACK,
			name:  g53.Root,
		}
		NULL_NODE.parent = NULL_NODE
		NULL_NODE.left = NULL_NODE
		NULL_NODE.right = NULL_NODE
		NULL_NODE.down = NULL_NODE
	}
}

func NewNode(name *g53.Name) *Node {
	return &Node{
		parent: NULL_NODE,
		left:   NULL_NODE,
		right:  NULL_NODE,
		color:  RED,
		down:   NULL_NODE,
		name:   name,
	}
}

func (node *Node) IsEmpty() bool {
	return node.data == nil
}

func (node *Node) IsLeaf() bool {
	return node.down == NULL_NODE
}

func (node *Node) GetFlag(flag RBNodeFlag) bool {
	return (node.flag & flag) != 0
}

func (node *Node) SetFlag(flag RBNodeFlag, set bool) {
	if set {
		node.flag = node.flag | flag
	} else {
		node.flag = node.flag & (^flag)
	}
}

func (node *Node) successor() *Node {
	current := node
	if node.right != NULL_NODE {
		current = node.right
		for current.left != NULL_NODE {
			current = current.left
		}
		return current
	}

	// Otherwise go up until we find the first left branch on our path to
	// root.  If found, the parent of the branch is the successor.
	// Otherwise, we return the null node
	parent := current.parent
	for parent != NULL_NODE && current == parent.right {
		current = parent
		parent = parent.parent
	}
	return parent
}

func (node *Node) SetData(data interface{}) {
	node.data = data
}

func (node *Node) Data() interface{} {
	return node.data
}

func (node *Node) Name() *g53.Name {
	return node.name
}

func (node *Node) Clean() {
	if node == NULL_NODE {
		return
	}

	if node.left != NULL_NODE {
		node.left.Clean()
		node.left = NULL_NODE
	}

	if node.right != NULL_NODE {
		node.right.Clean()
		node.right = NULL_NODE
	}

	if node.down != NULL_NODE {
		node.down.Clean()
		node.down = NULL_NODE
	}

	node.parent = NULL_NODE
	node.name = nil
	node.data = nil
}

type ValueCloneFunc func(interface{}) interface{}

func DefaultValueCloneFunc(v interface{}) interface{} {
	return v
}

func (n *Node) Clone(valueConeFunc ValueCloneFunc) *Node {
	if n == NULL_NODE {
		return NULL_NODE
	}

	new := *n
	if new.data != nil {
		new.data = valueConeFunc(new.data)
	}
	new.left = new.left.Clone(valueConeFunc)
	new.right = new.right.Clone(valueConeFunc)
	if new.left != NULL_NODE {
		new.left.parent = &new
	}
	if new.right != NULL_NODE {
		new.right.parent = &new
	}
	new.down = new.down.Clone(valueConeFunc)
	return &new
}
