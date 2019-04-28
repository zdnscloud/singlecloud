package memoryzone

import (
	"net"
	"sync"
	"time"

	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/vanguard/acl"
	"github.com/zdnscloud/vanguard/logger"
	"github.com/zdnscloud/vanguard/resolver/auth/zone"
)

const (
	cmdChanBuffer      = 128
	emptyNodeWaterMark = 40 // 40%
	vaccumInterval     = 5 * time.Second
)

type memoryTx struct {
	owner *DynamicZone
	tmp   *MemoryZone
	lock  *sync.RWMutex
}

func (tx *memoryTx) Commit() error {
	defer tx.lock.Unlock()

	if err := tx.tmp.validate(); err != nil {
		return err
	}

	old := tx.owner.MemoryZone
	tx.owner.MemoryZone = tx.tmp
	tx.tmp = nil
	go old.clean()
	return nil
}

func (tx *memoryTx) RollBack() error {
	tx.lock.Unlock()
	old := tx.tmp
	tx.tmp = nil
	go old.clean()
	return nil
}

type emptyZoneFinderCtx struct {
	result zone.FindResult
}

func (ctx *emptyZoneFinderCtx) GetResult() *zone.FindResult {
	return &ctx.result
}

func (ctx *emptyZoneFinderCtx) GetAdditional() []*g53.RRset {
	return nil
}

type DynamicZone struct {
	*MemoryZone
	stopCh  chan struct{}
	lock    sync.RWMutex
	masters []string
	acls    []string
}

func NewDynamicZone(origin *g53.Name) *DynamicZone {
	dz := &DynamicZone{
		MemoryZone: newMemoryZone(origin),
		stopCh:     make(chan struct{}),
		acls:       nil,
	}
	go dz.vaccumRoutine()
	return dz
}

func (z *DynamicZone) Load(loadChan <-chan *g53.RRset, abortChan <-chan struct{}) error {
	newMemZone := newMemoryZone(z.MemoryZone.origin)
	rrCount := 0
	isClose := false
	for {
		select {
		case <-abortChan:
			return zone.ErrAbortLoad
		case rrset, ok := <-loadChan:
			if ok {
				if err := newMemZone.addRRset(rrset); err != nil {
					if err == zone.ErrNoEffectiveUpdate {
						logger.GetLogger().Warn("ignore rr:%s %s %s", rrset.Name.String(false),
							rrset.Type.String(), rrset.Rdatas[0].String())
					} else {
						return err
					}
				} else {
					rrCount += 1
				}
			} else {
				isClose = true
			}
		}
		if isClose {
			break
		}
	}

	if err := newMemZone.validate(); err != nil {
		return err
	}

	logger.GetLogger().Info("load %d rrs in zone %s from master server", rrCount, z.origin.String(false))

	z.lock.Lock()
	z.MemoryZone = newMemZone
	z.lock.Unlock()

	return nil
}

func (z *DynamicZone) Dump() ([]*g53.RRset, error) {
	return z.MemoryZone.dump()
}

func (z *DynamicZone) GetUpdator(ip net.IP, force bool) (zone.ZoneUpdator, bool) {
	if force {
		return z, true
	}

	z.lock.RLock()
	defer z.lock.RUnlock()
	if len(z.masters) != 0 {
		return nil, false
	}

	if ip == nil || len(z.acls) == 0 {
		return z, true
	} else {
		for _, aclName := range z.acls {
			if acl.GetAclManager().Find(aclName, ip) {
				return z, true
			}
		}
		return nil, false
	}
}

func (z *DynamicZone) IsMaster() bool {
	z.lock.RLock()
	defer z.lock.RUnlock()
	return len(z.masters) == 0
}

func (z *DynamicZone) SetMasters(masters []string) {
	z.lock.Lock()
	defer z.lock.Unlock()
	z.masters = masters
}

func (z *DynamicZone) Masters() []string {
	z.lock.RLock()
	defer z.lock.RUnlock()
	if z.IsMaster() {
		return nil
	} else {
		masters := make([]string, len(z.masters))
		copy(masters, z.masters)
		return masters
	}
}

func (z *DynamicZone) SetAcls(acls []string) {
	z.lock.Lock()
	z.acls = acls
	z.lock.Unlock()
}

func (z *DynamicZone) Find(name *g53.Name, typ g53.RRType, option zone.FindOption) zone.FinderContext {
	if z.MemoryZone.isEmpty() {
		return &emptyZoneFinderCtx{
			result: zone.FindResult{Type: zone.FRServFail},
		}
	} else {
		z.lock.RLock()
		ctx := z.MemoryZone.find(name, typ, option)
		z.lock.RUnlock()

		if ctx.result.RRset != nil {
			z.lock.Lock()
			ctx.result.RRset.RotateRdata()
			z.lock.Unlock()
		}

		return &dynamicZoneFinderCtx{
			memoryZoneFinderCtx: ctx,
			zone:                z,
		}
	}
}

func (z *DynamicZone) Begin() (zone.Transaction, error) {
	z.lock.Lock()
	return &memoryTx{
		lock:  &z.lock,
		owner: z,
		tmp:   z.MemoryZone.clone(),
	}, nil
}

func (z *DynamicZone) Add(tx zone.Transaction, rrset *g53.RRset) error {
	return tx.(*memoryTx).tmp.addRRset(rrset)
}

func (z *DynamicZone) DeleteRRset(tx zone.Transaction, rrset *g53.RRset) error {
	_, err := tx.(*memoryTx).tmp.deleteRRset(rrset)
	return err
}

func (z *DynamicZone) DeleteDomain(tx zone.Transaction, name *g53.Name) error {
	_, err := tx.(*memoryTx).tmp.deleteDomain(name)
	return err
}

func (z *DynamicZone) DeleteRr(tx zone.Transaction, rrset *g53.RRset) error {
	_, err := tx.(*memoryTx).tmp.deleteRr(rrset)
	return err
}

func (z *DynamicZone) IncreaseSerialNumber(tx zone.Transaction) {
	tx.(*memoryTx).tmp.increaseSerialNumber()
}

func (z *DynamicZone) GetOrigin() *g53.Name {
	return z.MemoryZone.getOrigin()
}

func (z *DynamicZone) DomainCount() int {
	z.lock.Lock()
	defer z.lock.Unlock()
	return z.MemoryZone.domains.NodeCount()
}

func (z *DynamicZone) vaccumRoutine() {
	timer := time.NewTicker(vaccumInterval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
		case <-z.stopCh:
			z.stopCh <- struct{}{}
			return
		}
		z.removeEmptyNodes()
	}
}

func (z *DynamicZone) removeEmptyNodes() {
	z.lock.RLock()
	if z.MemoryZone.emptyNodeRatio() < emptyNodeWaterMark || z.MemoryZone.isEmpty() {
		z.lock.RUnlock()
		return
	}
	z.lock.RUnlock()

	z.lock.Lock()
	z.MemoryZone.removeEmptyNode()
	z.lock.Unlock()
}

type dynamicZoneFinderCtx struct {
	*memoryZoneFinderCtx
	zone *DynamicZone
}

func (ctx *dynamicZoneFinderCtx) GetAdditional() []*g53.RRset {
	ctx.zone.lock.RLock()
	rrsets := ctx.memoryZoneFinderCtx.GetAdditional()
	ctx.zone.lock.RUnlock()

	if len(rrsets) > 0 {
		ctx.zone.lock.Lock()
		for _, rrset := range rrsets {
			rrset.RotateRdata()
		}
		ctx.zone.lock.Unlock()
	}
	return rrsets
}

func (z *DynamicZone) Clean() error {
	z.stopCh <- struct{}{}
	<-z.stopCh
	return nil
}
