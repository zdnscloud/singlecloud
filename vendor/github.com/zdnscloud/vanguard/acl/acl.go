package acl

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/zdnscloud/cement/netradix"
	"github.com/zdnscloud/vanguard/config"
)

type Acl struct {
	tree         *netradix.NetRadixTree
	take_effect  uint32
	valid_time   *Schedule
	invalid_time *Schedule
}

func NewAcl(ips []string, validInterval, invalidInterval []config.TimeRange) (*Acl, error) {
	acl := &Acl{
		take_effect: 1,
		tree:        netradix.NewNetRadixTree(),
	}

	for _, ip := range ips {
		if err := acl.tree.Add(ip, struct{}{}); err != nil {
			return nil, fmt.Errorf("address %s isn't valid: %s", ip, err.Error())
		}
	}

	var err error
	if len(validInterval) > 0 {
		acl.valid_time, err = newSchedule(validInterval)
		if err != nil {
			return nil, err
		}
	}

	if len(invalidInterval) > 0 {
		acl.invalid_time, err = newSchedule(invalidInterval)
		if err != nil {
			return nil, err
		}
	}

	acl.CheckValid(time.Now())
	return acl, nil
}

func (a *Acl) Include(ip net.IP) bool {
	if atomic.LoadUint32(&a.take_effect) == 0 {
		return false
	}

	_, found := a.tree.SearchBest(ip)
	return found
}

func (a *Acl) CheckValid(now time.Time) {
	if a.invalid_time == nil && a.valid_time == nil {
		return
	}

	if (a.invalid_time != nil && a.invalid_time.IncludeTime(now)) ||
		(a.valid_time != nil && a.valid_time.IncludeTime(now) == false) {
		a.SetValid(false)
	} else {
		a.SetValid(true)
	}
}

func (a *Acl) SetValid(valid bool) {
	if valid {
		atomic.CompareAndSwapUint32(&a.take_effect, 0, 1)
	} else {
		atomic.CompareAndSwapUint32(&a.take_effect, 1, 0)
	}
}
