package domaintree

import (
	"errors"
	"fmt"
	"github.com/zdnscloud/g53"
)

type SearchResult int

const (
	ExactMatch   SearchResult = 0
	PartialMatch SearchResult = 1
	NotFound     SearchResult = 2
)

var ErrAlreadyExist = errors.New("name already exists")

type DomainTree struct {
	returnEmptyNode bool
	root            *Node
	nodeCount       int
}

type NodeCallBack func(*Node, interface{}) bool

func NewDomainTree(returnEmptyNode bool) *DomainTree {
	return &DomainTree{
		returnEmptyNode: returnEmptyNode,
		root:            NULL_NODE,
	}
}

func (tree *DomainTree) NodeCount() int {
	return tree.nodeCount
}

func (tree *DomainTree) Search(name *g53.Name) (*Node, SearchResult) {
	nodePath := NewNodeChain()
	return tree.SearchExt(name, nodePath, nil, nil)
}

func (tree *DomainTree) Clean() {
	tree.root.Clean()
	tree.root = NULL_NODE
	tree.nodeCount = 0
}

func (tree *DomainTree) SearchExt(name *g53.Name, nodePath *NodeChain, callback NodeCallBack, params interface{}) (*Node, SearchResult) {
	if nodePath.IsEmpty() == false {
		panic("search is given a null empty chain")
	}

	var target *Node
	node := tree.root
	ret := NotFound
	for node != NULL_NODE {
		nodePath.lastCompared = node
		comparison := name.Compare(node.name, false)
		nodePath.lastComparison = comparison
		if comparison.Relation == g53.EQUAL {
			if tree.returnEmptyNode || node.IsEmpty() == false {
				nodePath.push(node)
				target = node
				ret = ExactMatch
			}
			break
		} else {
			commonLabelCount := comparison.CommonLabelCount
			// If the common label count is 1, there is no common label between
			// the two names, except the trailing "dot".
			if commonLabelCount == 1 && node.name.IsRoot() == false {
				if comparison.Order < 0 {
					node = node.left
				} else {
					node = node.right
				}
			} else if comparison.Relation == g53.SUBDOMAIN {
				if tree.returnEmptyNode || node.IsEmpty() == false {
					ret = PartialMatch
					target = node
					if callback != nil && node.GetFlag(NF_CALLBACK) {
						if callback(node, params) {
							break
						}
					}
				}
				nodePath.push(node)
				name, _ = name.Subtract(node.name)
				node = node.down
			} else {
				break
			}
		}
	}
	return target, ret
}

func (tree *DomainTree) nextNode(nodePath *NodeChain) *Node {
	if nodePath.IsEmpty() {
		panic("next node is given a empty node path")
	}

	node := nodePath.Top()
	if node.down != NULL_NODE {
		leftMost := node.down
		for leftMost.left != NULL_NODE {
			leftMost = leftMost.left
		}
		nodePath.push(leftMost)
		return (leftMost)
	}

	// node_path go to up level
	nodePath.Pop()
	// otherwise found the successor node in current level
	successor := node.successor()
	if successor != NULL_NODE {
		nodePath.push(successor)
		return successor
	}

	// if no successor found move to up level, the next successor
	// is the successor of up node in the up level tree, if
	// up node doesn't have successor we gonna keep moving to up
	// level
	for nodePath.IsEmpty() == false {
		upNodeSuccessor := nodePath.Top().successor()
		nodePath.Pop()
		if upNodeSuccessor != NULL_NODE {
			nodePath.push(upNodeSuccessor)
			return upNodeSuccessor
		}
	}

	return nil
}

func (tree *DomainTree) Insert(name *g53.Name) (*Node, error) {
	if name.IsRoot() {
		return tree.insertRoot()
	}

	parent := NULL_NODE
	upNode := NULL_NODE
	current := tree.root

	order := -1
	for current != NULL_NODE {
		comparison := name.Compare(current.name, false)
		if comparison.Relation == g53.EQUAL {
			if current.IsEmpty() {
				return current, nil
			} else {
				return current, ErrAlreadyExist
			}
		} else {
			if comparison.CommonLabelCount == 1 && current.name.IsRoot() == false {
				parent = current
				order = comparison.Order
				if order < 0 {
					current = current.left
				} else {
					current = current.right
				}
			} else {
				// insert sub domain to sub tree
				if comparison.Relation == g53.SUBDOMAIN {
					parent = NULL_NODE
					upNode = current
					name, _ = name.Subtract(current.name)
					current = current.down
				} else {
					// The number of labels in common is fewer
					// than the number of labels at the current
					// node, so the current node must be adjusted
					// to have just the common suffix, and a down
					// pointer made to a new tree.
					commonAncestor, _ := name.Split(
						name.LabelCount()-uint(comparison.CommonLabelCount),
						uint(comparison.CommonLabelCount))
					tree.nodeFission(current, commonAncestor)
				}
			}
		}
	}

	currentRoot := &tree.root
	if upNode != NULL_NODE {
		currentRoot = &upNode.down
	}
	node := NewNode(name)
	node.parent = parent
	if parent == NULL_NODE {
		*currentRoot = node
		node.color = BLACK
	} else if order < 0 {
		parent.left = node
	} else {
		parent.right = node
	}
	tree.insertRebalance(currentRoot, node)
	tree.nodeCount += 1
	return node, nil
}

func (tree *DomainTree) insertRoot() (*Node, error) {
	current := tree.root
	if current != NULL_NODE && current.name.IsRoot() {
		return current, nil
	}

	node := NewNode(g53.Root)
	node.color = BLACK
	node.down = current
	if current != NULL_NODE {
		current.parent = node
		current.color = BLACK
	}
	tree.root = node
	tree.nodeCount += 1
	return node, nil
}

func (tree *DomainTree) nodeFission(oldNode *Node, baseName *g53.Name) {
	oldName := oldNode.name
	subName, _ := oldName.Subtract(baseName)
	downNode := NewNode(subName)
	oldNode.name = baseName
	downNode.data, oldNode.data = oldNode.data, downNode.data
	downNode.flag, oldNode.flag = oldNode.flag, downNode.flag
	downNode.down = oldNode.down
	oldNode.down = downNode
	downNode.color = BLACK
	tree.nodeCount += 1
}

func (tree *DomainTree) insertRebalance(root **Node, node *Node) {
	var uncle *Node
	for node != *root && node.parent.color == RED {
		if node.parent == node.parent.parent.left {
			uncle = node.parent.parent.right
			if uncle.color == RED {
				node.parent.color = BLACK
				uncle.color = BLACK
				node.parent.parent.color = RED
				node = node.parent.parent
			} else {
				if node == node.parent.right {
					node = node.parent
					tree.leftRotate(root, node)
				}
				node.parent.color = BLACK
				node.parent.parent.color = RED
				tree.rightRotate(root, node.parent.parent)
			}
		} else {
			uncle = node.parent.parent.left
			if uncle.color == RED {
				node.parent.color = BLACK
				uncle.color = BLACK
				node.parent.parent.color = RED
				node = node.parent.parent
			} else {
				if node == node.parent.left {
					node = node.parent
					tree.rightRotate(root, node)
				}
				node.parent.color = BLACK
				node.parent.parent.color = RED
				tree.leftRotate(root, node.parent.parent)
			}
		}
	}

	(*root).color = BLACK
}

func (tree *DomainTree) leftRotate(root **Node, node *Node) *Node {
	right := node.right
	node.right = right.left
	if right.left != NULL_NODE {
		right.left.parent = node
	}
	right.parent = node.parent
	if node.parent != NULL_NODE {
		if node == node.parent.left {
			node.parent.left = right
		} else {
			node.parent.right = right
		}
	} else {
		*root = right
	}

	right.left = node
	node.parent = right
	return node
}

func (tree *DomainTree) rightRotate(root **Node, node *Node) *Node {
	left := node.left
	node.left = left.right
	if left.right != NULL_NODE {
		left.right.parent = node
	}
	left.parent = node.parent
	if node.parent != NULL_NODE {
		if node == node.parent.right {
			node.parent.right = left
		} else {
			node.parent.left = left
		}
	} else {
		*root = left
	}
	left.right = node
	node.parent = left
	return node
}

func (tree *DomainTree) Dump(depth int) {
	tree.indent(depth)
	fmt.Printf("tree has %d node(s)\n", tree.nodeCount)
	tree.dumpTreeHelper(tree.root, depth)
}

func (tree *DomainTree) dumpTreeHelper(node *Node, depth int) {
	if node == NULL_NODE {
		tree.indent(depth)
		fmt.Printf("NULL\n")
		return
	}

	tree.indent(depth)
	fmt.Printf("%s (%s)", node.name.String(false), node.color.String())
	if node.IsEmpty() {
		fmt.Printf("[invisible] \n")
	} else {
		fmt.Printf("\n")
	}

	if node.down != NULL_NODE {
		tree.indent(depth + 1)
		fmt.Printf("begin down from %s\n", node.name.String(false))
		tree.dumpTreeHelper(node.down, depth+1)
		tree.indent(depth + 1)
		fmt.Printf("end down from %s\n", node.name.String(false))
	}
	tree.dumpTreeHelper(node.left, depth+1)
	tree.dumpTreeHelper(node.right, depth+1)
}

const INDENT_FOR_EACH_DEPTH = 5

func (tree *DomainTree) indent(depth int) {
	spaceLen := depth * INDENT_FOR_EACH_DEPTH
	space := make([]byte, spaceLen)
	for i := 0; i < spaceLen; i++ {
		space[i] = byte(' ')
	}
	fmt.Printf("%s", string(space))
}

func (tree *DomainTree) ForEach(fn func(*Node)) {
	tree.forEachHelper(tree.root, tree.returnEmptyNode, fn)
}

func (tree *DomainTree) forEachHelper(node *Node, returnEmptyNode bool, fn func(*Node)) {
	if node == NULL_NODE {
		return
	}

	if returnEmptyNode || node.IsEmpty() == false {
		fn(node)
	}

	tree.forEachHelper(node.left, returnEmptyNode, fn)
	tree.forEachHelper(node.right, returnEmptyNode, fn)
	tree.forEachHelper(node.down, returnEmptyNode, fn)
}

func (tree *DomainTree) ForEachEx(fn func(*g53.Name, *Node)) {
	tree.forEachExHelper(tree.root, g53.Root, tree.returnEmptyNode, fn)
}

func (tree *DomainTree) forEachExHelper(node *Node, parentFullName *g53.Name, returnEmptyNode bool, fn func(*g53.Name, *Node)) {
	if node == NULL_NODE {
		return
	}

	if returnEmptyNode || node.IsEmpty() == false {
		newParent, _ := node.name.Concat(parentFullName)
		fn(newParent, node)
	}

	tree.forEachExHelper(node.left, parentFullName, returnEmptyNode, fn)
	tree.forEachExHelper(node.right, parentFullName, returnEmptyNode, fn)
	newParent, _ := node.name.Concat(parentFullName)
	tree.forEachExHelper(node.down, newParent, returnEmptyNode, fn)
}

func (tree *DomainTree) IsNodeNonTerminal(node *Node) bool {
	return tree.anyHelper(node.down, func(n *Node) bool {
		return n.IsEmpty() == false
	})
}

func (tree *DomainTree) All(fn func(*Node) bool) bool {
	return tree.allHelper(tree.root, fn)
}

func (tree *DomainTree) allHelper(node *Node, fn func(*Node) bool) bool {
	if node == NULL_NODE {
		return true
	}

	if tree.returnEmptyNode || node.IsEmpty() == false {
		if fn(node) == false {
			return false
		}
	}

	return tree.allHelper(node.left, fn) &&
		tree.allHelper(node.right, fn) &&
		tree.allHelper(node.down, fn)
}

func (tree *DomainTree) Any(fn func(*Node) bool) bool {
	return tree.anyHelper(tree.root, fn)
}

func (tree *DomainTree) anyHelper(node *Node, fn func(*Node) bool) bool {
	if node == NULL_NODE {
		return false
	}

	if tree.returnEmptyNode || node.IsEmpty() == false {
		if fn(node) == true {
			return true
		}
	}

	return tree.anyHelper(node.left, fn) ||
		tree.anyHelper(node.right, fn) ||
		tree.anyHelper(node.down, fn)
}

func (tree *DomainTree) EmptyLeafNodeRatio() int {
	if tree.nodeCount == 0 {
		return 0
	}

	emptyNodeCount := 0
	tree.forEachHelper(tree.root, true, func(node *Node) {
		if node.IsEmpty() && node.IsLeaf() {
			emptyNodeCount += 1
		}
	})
	return (emptyNodeCount * 100 / tree.nodeCount)
}

func (tree *DomainTree) RemoveEmptyLeafNode() *DomainTree {
	new := NewDomainTree(tree.returnEmptyNode)
	tree.forEachExHelper(tree.root, g53.Root, true, func(name *g53.Name, node *Node) {
		if node.IsEmpty() == false || node.IsLeaf() == false {
			n, _ := new.Insert(name)
			n.SetData(node.Data())
		}
	})
	return new
}

func (tree *DomainTree) Clone(valueConeFunc ValueCloneFunc) *DomainTree {
	if valueConeFunc == nil {
		valueConeFunc = DefaultValueCloneFunc
	}

	new := NewDomainTree(tree.returnEmptyNode)
	new.root = tree.root.Clone(valueConeFunc)
	new.nodeCount = tree.nodeCount
	return new
}
