// LRU cache for dns with ttl
package cache

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

const (
	// only cache hit count large than this should do prefetch
	prefetchMinHitCount = 10
	// only cache will be expired in this time should do prefetch
	prefetchLeftTTL = 10 * time.Second
)

// entry represent an item in LRU.deque.
// key is used when delete item from LRU.items
type entry struct {
	key   interface{}
	value interface{}
	hit int
	ttl   time.Time
}

func (e *entry) toItem() Item {
	return Item{key: e.key.(string), hit: e.hit, ttl: e.ttl}
}

type Item struct {
	key string
	hit int
	ttl time.Time
}

type LRU struct {
	capacity int
	deque    *list.List
	items    map[interface{}]*list.Element
	lock     *sync.Mutex
}

func NewLRU(capacity int) (*LRU, error) {
	if capacity <= 0 {
		return nil, errors.New("LRU capacity must > 0")
	}
	return &LRU{
		capacity: capacity,
		deque:    list.New(),
		items:    make(map[interface{}]*list.Element),
		lock:     new(sync.Mutex),
	}, nil
}

func (c *LRU) PrefetchTopN(n int) []Item {
	item := c.deque.Front()
	result := make([]Item, 0, n)
	count := 0
	for item != nil && count <= n {
		v := item.Value.(*entry)
		if shouldPrefetch(v, prefetchMinHitCount, prefetchLeftTTL) {
			result = append(result, v.toItem())
		}
		item = item.Next()
		count++
	}
	return result
}

func  shouldPrefetch(e *entry, minHit int, leftTTL time.Duration) bool {
	if e.hit >= minHit && time.Now().Sub(e.ttl) <= leftTTL {
		return true
	}
	return false
}


func (c *LRU) Get(key interface{}) interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()

	if v, ok := c.items[key]; ok {
		_v := v.Value.(*entry)
		if _v.ttl.Before(time.Now()) {
			// expired
			c.removeElement(v)
			return nil
		}
		c.deque.MoveToFront(v)
		_v.hit++
		return _v.value
	}
	return nil
}

func (c *LRU) Add(key, value interface{}, ttl time.Time) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if v, ok := c.items[key]; ok {
		_v := v.Value.(*entry)
		_v.value = value
		_v.ttl = ttl
		_v.hit++
		c.deque.MoveToFront(v)
		return false
	}
	ent := &entry{key, value, 1, ttl}
	ele := c.deque.PushFront(ent)
	c.items[key] = ele
	if c.Len() > c.capacity {
		ele := c.deque.Back()
		c.removeElement(ele)
	}
	return true
}

func (c *LRU) removeElement(item *list.Element) {
	c.deque.Remove(item)
	delete(c.items, item.Value.(*entry).key)
}

func (c *LRU) Len() int {
	return c.deque.Len()
}
