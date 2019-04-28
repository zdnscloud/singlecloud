package metrics

import (
	"sync/atomic"
)

type Counter struct {
	count uint64
}

func newCounter() *Counter {
	return &Counter{
		count: 0,
	}
}

func (c *Counter) Clear() {
	atomic.StoreUint64(&c.count, 0)
}

func (c *Counter) Count() uint64 {
	return atomic.LoadUint64(&c.count)
}

func (c *Counter) Inc() {
	atomic.AddUint64(&c.count, 1)
}
