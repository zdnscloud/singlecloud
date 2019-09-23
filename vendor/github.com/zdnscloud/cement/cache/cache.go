package cache

import (
	"container/list"
	"sync"
	"time"
)

type Key uint64
type Value interface{}
type Hasher func(Value) Key

type listEntry struct {
	value      Value
	expireTime time.Time
}

type Cache struct {
	maxMessageCount uint32
	OnEvicted       func(interface{})

	ll     *list.List
	cache  map[Key]*list.Element
	hasher Hasher
	lock   sync.Locker
}

func New(maxMessageCount uint32, hasher Hasher, threadSafe bool) *Cache {
	var lock sync.Locker
	if threadSafe {
		lock = &sync.Mutex{}
	} else {
		lock = &DumbLock{}
	}

	return &Cache{
		maxMessageCount: maxMessageCount,
		ll:              list.New(),
		cache:           make(map[Key]*list.Element),
		hasher:          hasher,
		lock:            lock,
	}
}

func (c *Cache) Add(value Value, ttl time.Duration) {
	if c.maxMessageCount == 0 {
		return
	}

	key := c.hasher(value)
	entry := &listEntry{
		value:      value,
		expireTime: time.Now().Add(ttl),
	}

	c.lock.Lock()
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		elem.Value = entry
	} else {
		elem := c.ll.PushFront(entry)
		c.cache[key] = elem
		if uint32(c.ll.Len()) > c.maxMessageCount {
			c.removeOldest()
		}
	}
	c.lock.Unlock()
}

func (c *Cache) Shrink() {
	c.lock.Lock()
	for uint32(c.ll.Len()) > c.maxMessageCount {
		c.removeOldest()
	}
	c.lock.Unlock()
}

func (c *Cache) Get(key Key) (Value, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if elem, hit := c.cache[key]; hit {
		c.ll.MoveToFront(elem)
		entry := elem.Value.(*listEntry)
		if entry.expireTime.After(time.Now()) {
			return entry.value, true
		}
	}
	return nil, false
}

func (c *Cache) Remove(key Key) {
	c.lock.Lock()
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
	c.lock.Unlock()
}

func (c *Cache) removeOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *Cache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	value := e.Value.(*listEntry).value
	key := c.hasher(value)
	delete(c.cache, key)
	if c.OnEvicted != nil {
		c.OnEvicted(value)
	}
}

func (c *Cache) Len() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.ll.Len()
}

func (c *Cache) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.ll.Init()
	c.cache = make(map[Key]*list.Element)
}
