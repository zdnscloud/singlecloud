package domaintree

import (
	"errors"
	"sync"

	"github.com/zdnscloud/g53"
	dt "github.com/zdnscloud/g53/domaintree"
)

type SearchResult int

var ErrInsertNilValue = errors.New("insert nil value")

const (
	ExactMatch      SearchResult = 0
	ClosestEncloser SearchResult = 1
	NotFound        SearchResult = 2
)

type DomainTree struct {
	nodes *dt.DomainTree
	lock  RWLocker
}

func NewDomainTree() *DomainTree {
	return New(false)
}

func NewSafeDomainTree() *DomainTree {
	return New(true)
}

func New(threadSafe bool) *DomainTree {
	var lock RWLocker = &sync.RWMutex{}
	if threadSafe == false {
		lock = &DumbRWMutex{}
	}

	return &DomainTree{
		nodes: dt.NewDomainTree(false),
		lock:  lock,
	}
}

func (t *DomainTree) Search(name *g53.Name) (*g53.Name, interface{}, SearchResult) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	nodePath := dt.NewNodeChain()
	node, ret := t.nodes.SearchExt(name, nodePath, nil, nil)
	switch ret {
	case dt.ExactMatch:
		return name, node.Data(), ExactMatch
	case dt.PartialMatch:
		for nodePath.IsEmpty() == false {
			if nodePath.Top() == node {
				break
			}
			nodePath.Pop()
		}
		return nodePath.GetAbsoluteName(), node.Data(), ClosestEncloser
	case dt.NotFound:
		for nodePath.IsEmpty() == false {
			parent := nodePath.Top()
			if parent.IsEmpty() == false && name.IsSubDomain(parent.Name()) {
				return nodePath.GetAbsoluteName(), parent.Data(), ClosestEncloser
			} else {
				nodePath.Pop()
			}
		}
		return nil, nil, NotFound
	default:
		panic("search should return other result")
	}
}

func (t *DomainTree) SearchParents(name *g53.Name) (*dt.NodeChain, SearchResult) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	nodePath := dt.NewNodeChain()
	node, ret := t.nodes.SearchExt(name, nodePath, nil, nil)
	switch ret {
	case dt.ExactMatch:
		return nodePath, ExactMatch
	case dt.PartialMatch:
		for nodePath.IsEmpty() == false {
			if nodePath.Top() == node {
				break
			}
			nodePath.Pop()
		}
		return nodePath, ClosestEncloser
	case dt.NotFound:
		for nodePath.IsEmpty() == false {
			parent := nodePath.Top()
			if parent.IsEmpty() == false && name.IsSubDomain(parent.Name()) {
				return nodePath, ClosestEncloser
			} else {
				nodePath.Pop()
			}
		}
		return nil, NotFound
	default:
		panic("search should return other result")
	}
}

func (t *DomainTree) Insert(name *g53.Name, data interface{}) (*dt.Node, error) {
	if data == nil {
		return nil, ErrInsertNilValue
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	node, err := t.nodes.Insert(name)
	if err != nil && err != dt.ErrAlreadyExist {
		return nil, err
	} else {
		node.SetData(data)
		return node, nil
	}
}

func (t *DomainTree) Delete(name *g53.Name) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.nodes.Remove(name)
}

func (t *DomainTree) ForEach(f func(interface{})) {
	t.nodes.ForEach(func(n *dt.Node) {
		f(n.Data())
	})
}
