// LRU cache for dns with ttl
package main

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

// entry represent an item in LRU.deque.
// key is used when delete item from LRU.items
type entry struct {
	key   interface{}
	value interface{}
	ttl   time.Time
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

func (c *LRU) Get(key interface{}) interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()

	if v, ok := c.items[key]; ok {
		if v.Value.(*entry).ttl.Before(time.Now()) {
			// expired
			c.removeElement(v)
			return nil
		}
		c.deque.MoveToFront(v)
		return v.Value.(*entry).value
	}
	return nil
}

func (c *LRU) Add(key, value interface{}, ttl time.Time) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if v, ok := c.items[key]; ok {
		v.Value.(*entry).value = value
		v.Value.(*entry).ttl = ttl
		c.deque.MoveToFront(v)
		return false
	}
	ent := &entry{key, value, ttl}
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
