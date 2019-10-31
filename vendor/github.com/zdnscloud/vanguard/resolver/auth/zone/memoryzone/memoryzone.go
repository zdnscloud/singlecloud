package memoryzone

import (
	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/g53/domaintree"
	"github.com/zdnscloud/vanguard/resolver/auth/zone"
)

const wildcardMark = domaintree.NF_USER1

type NameNode map[g53.RRType]*g53.RRset

func (n NameNode) clone() NameNode {
	new := make(map[g53.RRType]*g53.RRset)
	for t, rrset := range n {
		new[t] = rrset
	}
	return new
}

type MemoryZone struct {
	origin     *g53.Name
	originNode *domaintree.Node
	domains    *domaintree.DomainTree
}

type memoryZoneFinderCtx struct {
	result zone.FindResult
	node   NameNode
	finder *MemoryZone
}

func (ctx *memoryZoneFinderCtx) GetResult() *zone.FindResult {
	return &ctx.result
}

func (ctx *memoryZoneFinderCtx) GetAdditional() []*g53.RRset {
	rrset := ctx.result.RRset
	addrs := []*g53.RRset{}
	if rrset != nil {
		for _, rdata := range rrset.Rdatas {
			if rrset.Type == g53.RR_NS {
				addrs = append(addrs, ctx.finder.getAdditioanlAddrs(rdata.(*g53.NS).Name)...)
			} else if rrset.Type == g53.RR_MX {
				addrs = append(addrs, ctx.finder.getAdditioanlAddrs(rdata.(*g53.MX).Exchange)...)
			} else if rrset.Type == g53.RR_SRV {
				addrs = append(addrs, ctx.finder.getAdditioanlAddrs(rdata.(*g53.SRV).Target)...)
			}
		}
	}
	return addrs
}

type findState struct {
	zonecut NameNode
	rrset   *g53.RRset
	option  zone.FindOption
}

func newMemoryZone(origin *g53.Name) *MemoryZone {
	return newWithDomains(origin, domaintree.NewDomainTree(true))
}

func newWithDomains(origin *g53.Name, domains *domaintree.DomainTree) *MemoryZone {
	node, _ := domains.Insert(origin)
	return &MemoryZone{
		origin:     origin,
		originNode: node,
		domains:    domains,
	}
}

func (z *MemoryZone) getOrigin() *g53.Name {
	return z.origin
}

func zoneCutCallback(node *domaintree.Node, param interface{}) bool {
	nameNode := node.Data().(NameNode)
	ns, ok := nameNode[g53.RR_NS]
	if ok == false {
		panic("zone cut is called but no ns is found:" + node.Name().String(true))
	}

	state := param.(*findState)
	//top level zone cut must be hit
	if state.zonecut != nil {
		return false
	}

	state.zonecut = nameNode
	state.rrset = ns

	return state.option != zone.GlueOkFind
}

func (z *MemoryZone) dump() ([]*g53.RRset, error) {
	return nil, nil
}

func (z *MemoryZone) find(name *g53.Name, typ g53.RRType, option zone.FindOption) *memoryZoneFinderCtx {
	nodePath := domaintree.NewNodeChain()
	findState := &findState{
		option: option,
	}

	ctx := &memoryZoneFinderCtx{
		finder: z,
	}
	node, ret := z.domains.SearchExt(name, nodePath, zoneCutCallback, findState)
	switch ret {
	case domaintree.PartialMatch:
		if findState.zonecut != nil {
			ctx.result = zone.FindResult{
				Type:  zone.FRDelegation,
				RRset: findState.rrset,
			}
			ctx.node = findState.zonecut
			return ctx
		}

		if nodePath.LastComparison().Relation == g53.SUPERDOMAIN {
			ctx.result = zone.FindResult{Type: zone.FRNXRRset}
			return ctx
		}

		if node.GetFlag(wildcardMark) {
			wildcardName, _ := g53.NameFromStringUnsafe("*").Concat(nodePath.GetAbsoluteName())
			wildcard, ret := z.domains.Search(wildcardName)
			if ret != domaintree.ExactMatch {
				panic("wildcard isn't correct marked")
			}

			nameNode := wildcard.Data().(NameNode)
			if rrset, ok := nameNode[typ]; ok {
				synthesis := *rrset
				synthesis.Name = name
				ctx.result = zone.FindResult{
					Type:  zone.FRSuccess,
					RRset: &synthesis,
				}
				return ctx
			} else if cname, ok := nameNode[g53.RR_CNAME]; ok {
				synthesis := *cname
				synthesis.Name = name
				ctx.result = zone.FindResult{
					Type:  zone.FRCname,
					RRset: &synthesis,
				}
				return ctx
			} else {
				ctx.result = zone.FindResult{Type: zone.FRNXRRset}
				return ctx
			}
		}

		ctx.result = zone.FindResult{Type: zone.FRNXDomain}
		return ctx

	case domaintree.NotFound:
		ctx.result = zone.FindResult{Type: zone.FRNXDomain}
		return ctx
	}

	if node.IsEmpty() {
		if z.domains.IsNodeNonTerminal(node) {
			ctx.result = zone.FindResult{Type: zone.FRNXRRset}
		} else {
			ctx.result = zone.FindResult{Type: zone.FRNXDomain}
		}
		return ctx
	}

	nameNode := node.Data().(NameNode)
	ctx.node = nameNode
	if node.GetFlag(domaintree.NF_CALLBACK) && node != z.originNode {
		if ns, ok := nameNode[g53.RR_NS]; ok {
			ctx.result = zone.FindResult{
				Type:  zone.FRDelegation,
				RRset: ns,
			}
			return ctx
		}
	}

	if rrset, ok := nameNode[typ]; ok {
		ctx.result = zone.FindResult{
			Type:  zone.FRSuccess,
			RRset: rrset,
		}
		return ctx
	} else if cname, ok := nameNode[g53.RR_CNAME]; ok {
		ctx.result = zone.FindResult{
			Type:  zone.FRCname,
			RRset: cname,
		}
		return ctx
	} else {
		ctx.result = zone.FindResult{Type: zone.FRNXRRset}
		return ctx
	}
}

func (z *MemoryZone) getAdditioanlAddrs(name *g53.Name) []*g53.RRset {
	ctx := z.find(name, g53.RR_A, zone.GlueOkFind)
	addrs := []*g53.RRset{}
	result := ctx.GetResult()
	tryAAAA := false
	if result.Type == zone.FRSuccess {
		addrs = append(addrs, result.RRset)
		tryAAAA = true
	} else if result.Type == zone.FRNXRRset {
		tryAAAA = true
	}

	if tryAAAA {
		if aaaa, ok := ctx.node[g53.RR_AAAA]; ok {
			addrs = append(addrs, aaaa)
		}
	}
	return addrs
}

func (z *MemoryZone) addRRset(rrset *g53.RRset) error {
	if err := z.checkRRsetValid(rrset); err != nil {
		return err
	}

	node, err := z.domains.Insert(rrset.Name)
	if err != nil && err != domaintree.ErrAlreadyExist {
		return err
	}

	var nodeData NameNode
	if node.IsEmpty() {
		nodeData = make(NameNode)
		nodeData[rrset.Type] = rrset
		node.SetData(nodeData)
	} else {
		nodeData = node.Data().(NameNode)
		if err := z.checkRRsetCouldBeAdded(nodeData, rrset); err != nil {
			return err
		}

		if rrset.Type == g53.RR_SOA {
			oldSOA := nodeData[g53.RR_SOA]
			if oldSOA != nil && oldSOA.Rdatas[0].(*g53.SOA).Serial >= rrset.Rdatas[0].(*g53.SOA).Serial {
				return nil
			} else {
				nodeData[rrset.Type] = rrset
			}
		} else if oldRRset, ok := nodeData[rrset.Type]; ok {
			if rrset.Type == g53.RR_CNAME {
				nodeData[rrset.Type] = rrset
			} else {
				newRRset := oldRRset.Clone()
				allDuplicate := true
				for _, rdata := range rrset.Rdatas {
					if err := newRRset.AddRdata(rdata); err == nil {
						allDuplicate = false
					}
				}

				if allDuplicate && rrset.Ttl == newRRset.Ttl {
					return g53.ErrDuplicateRdata
				}
				newRRset.Ttl = rrset.Ttl
				nodeData[rrset.Type] = newRRset
			}
		} else {
			nodeData[rrset.Type] = rrset
		}
	}

	if rrset.Type == g53.RR_NS && rrset.Name.Equals(z.origin) == false {
		node.SetFlag(domaintree.NF_CALLBACK, true)
	}

	if rrset.Name.IsWildCard() {
		if err := z.markWildcardParent(rrset.Name); err != nil {
			return err
		}
	}

	return nil
}

func (z *MemoryZone) checkRRsetValid(rrset *g53.RRset) error {
	if rrset.Type == g53.RR_NS && rrset.Name.IsWildCard() {
		return zone.ErrNoEffectiveUpdate
	}

	if rrset.Name.IsSubDomain(z.origin) == false {
		return zone.ErrOutOfZone
	}

	return nil
}

func (z *MemoryZone) checkRRsetCouldBeAdded(nodeData NameNode, rrset *g53.RRset) error {
	if rrset.Type != g53.RR_CNAME {
		if _, ok := nodeData[g53.RR_CNAME]; ok {
			return zone.ErrCNAMECoExistsWithOtherRR
		}
	} else {
		if len(nodeData) > 1 {
			return zone.ErrCNAMECoExistsWithOtherRR
		} else if _, ok := nodeData[g53.RR_CNAME]; ok == false {
			return zone.ErrCNAMECoExistsWithOtherRR
		}
	}

	return nil
}

func (z *MemoryZone) isEmpty() bool {
	return z.originNode.Data() == nil
}

func (z *MemoryZone) validate() error {
	data := z.originNode.Data()
	if data == nil {
		return zone.ErrShortOfSOA
	}

	node := data.(NameNode)
	if _, ok := node[g53.RR_SOA]; ok == false {
		return zone.ErrShortOfSOA
	}

	if _, ok := node[g53.RR_NS]; ok == false {
		return zone.ErrShortOfNS
	}

	return nil
}

func (z *MemoryZone) markWildcardParent(name *g53.Name) error {
	parent, _ := name.Parent(1)
	node, err := z.domains.Insert(parent)

	if err != nil && err != domaintree.ErrAlreadyExist {
		return err
	}
	node.SetFlag(wildcardMark, true)
	return nil
}

func (z *MemoryZone) unmarkWildcardParent(name *g53.Name) {
	parent, _ := name.Parent(1)
	node, ret := z.domains.Search(parent)
	if ret != domaintree.ExactMatch {
		return
	}
	node.SetFlag(wildcardMark, false)
}

func (z *MemoryZone) checkRRsetCouldBeDeleted(rrset *g53.RRset) error {
	if rrset.Name.IsSubDomain(z.origin) == false {
		return zone.ErrOutOfZone
	}

	if rrset.Name.Equals(z.origin) {
		if rrset.Type == g53.RR_SOA {
			return zone.ErrShortOfSOA
		}

		if rrset.Type == g53.RR_NS {
			data := z.originNode.Data()
			leftNs := rdatasDiff(data.(NameNode)[g53.RR_NS].Rdatas, rrset.Rdatas)
			if len(leftNs) == 0 {
				return zone.ErrShortOfNS
			}
		}
	}

	return nil
}

func (z *MemoryZone) getNode(name *g53.Name) (*domaintree.Node, error) {
	node, ret := z.domains.Search(name)
	if ret != domaintree.ExactMatch {
		return nil, zone.ErrUnknownRRset
	}

	if node.IsEmpty() {
		return nil, zone.ErrUnknownRRset
	}
	return node, nil
}

func (z *MemoryZone) deleteNode(name *g53.Name) {
	z.domains.Remove(name)
	if name.IsWildCard() {
		z.unmarkWildcardParent(name)
	}
}

func (z *MemoryZone) deleteRRset(rrset *g53.RRset) (*domaintree.Node, error) {
	if err := z.checkRRsetCouldBeDeleted(rrset); err != nil {
		return nil, err
	}

	node, err := z.getNode(rrset.Name)
	if err != nil {
		return nil, err
	}

	nodeData := node.Data().(NameNode)
	_, ok := nodeData[rrset.Type]
	if ok == false {
		return nil, zone.ErrUnknownRRset
	}

	delete(nodeData, rrset.Type)
	if len(nodeData) == 0 {
		z.deleteNode(rrset.Name)
	}

	return node, nil
}

func (z *MemoryZone) deleteDomain(name *g53.Name) (*domaintree.Node, error) {
	if name.IsSubDomain(z.origin) == false {
		return nil, zone.ErrOutOfZone
	}

	node, err := z.getNode(name)
	if err != nil {
		return nil, err
	}

	if name.Equals(z.origin) {
		newData := make(NameNode)
		oldData := node.Data().(NameNode)
		newData[g53.RR_SOA] = oldData[g53.RR_SOA]
		newData[g53.RR_NS] = oldData[g53.RR_NS]
		node.SetData(newData)
	} else {
		z.deleteNode(name)
	}

	return node, nil
}

func (z *MemoryZone) deleteRr(rrset *g53.RRset) (*domaintree.Node, error) {
	if err := z.checkRRsetCouldBeDeleted(rrset); err != nil {
		return nil, err
	}

	node, err := z.getNode(rrset.Name)
	if err != nil {
		return nil, err
	}

	nodeData := node.Data().(NameNode)
	oldRRset, ok := nodeData[rrset.Type]
	if ok == false {
		return nil, zone.ErrUnknownRRset
	}

	leftRdatas := rdatasDiff(oldRRset.Rdatas, rrset.Rdatas)
	if len(leftRdatas) == 0 {
		delete(nodeData, rrset.Type)
		if len(nodeData) == 0 {
			z.deleteNode(rrset.Name)
		}
	} else {
		nodeData[rrset.Type] = &g53.RRset{
			Name:   oldRRset.Name,
			Type:   oldRRset.Type,
			Class:  oldRRset.Class,
			Ttl:    oldRRset.Ttl,
			Rdatas: leftRdatas,
		}
	}

	return node, nil
}

func (z *MemoryZone) clean() {
	z.originNode = domaintree.NULL_NODE
	z.domains.Clean()
}

func rdatasDiff(first []g53.Rdata, second []g53.Rdata) []g53.Rdata {
	var left []g53.Rdata
	for _, src := range first {
		inSecond := false
		for _, target := range second {
			if src.Compare(target) == 0 {
				inSecond = true
				break
			}
		}
		if inSecond == false {
			left = append(left, src)
		}
	}
	return left
}

func (z *MemoryZone) increaseSerialNumber() {
	data := z.originNode.Data()
	if data == nil {
		panic("zone short of soa")
	}

	soa := data.(NameNode)[g53.RR_SOA]
	if len(soa.Rdatas) != 1 {
		panic("zone soa rr isn't one")
	}

	soa.Rdatas[0].(*g53.SOA).Serial += 1
}

func cloneNode(v interface{}) interface{} {
	return v.(NameNode).clone()
}

func (z *MemoryZone) clone() *MemoryZone {
	return newWithDomains(z.origin, z.domains.Clone(cloneNode))
}
