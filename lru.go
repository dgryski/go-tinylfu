package tinylfu

import "container/list"

type lruItem struct {
	key   string
	value interface{}
}

// Cache is an LRU cache.  It is not safe for concurrent access.
type lruCache struct {
	data map[string]*list.Element
	cap  int
	ll   *list.List
}

func newLRU(cap int) *lruCache {
	return &lruCache{
		data: make(map[string]*list.Element),
		cap:  cap,
		ll:   list.New(),
	}
}

// Get returns a value from the cache
func (lru *lruCache) Get(key string) (interface{}, bool) {
	v, ok := lru.data[key]
	if !ok {
		return nil, false
	}

	item := v.Value.(*lruItem)
	lru.ll.MoveToFront(v)
	return item.value, true
}

// Set sets a value in the cache
func (lru *lruCache) Add(key string, value interface{}) (oldkey string, oldval interface{}, evicted bool) {
	if lru.ll.Len() < lru.cap {
		lru.data[key] = lru.ll.PushFront(&lruItem{key, value})
		return "", nil, false
	}

	// reuse the tail item
	e := lru.ll.Back()
	item := e.Value.(*lruItem)

	delete(lru.data, item.key)
	oldkey, oldval = item.key, item.value
	item.key = key
	item.value = value
	lru.data[key] = e
	lru.ll.MoveToFront(e)

	return oldkey, oldval, true
}

// Len returns the total number of items in the cache
func (lru *lruCache) Len() int {
	return len(lru.data)
}

// Remove removes an item from the cache, returning the item and a boolean indicating if it was found
func (lru *lruCache) Remove(key string) (interface{}, bool) {
	v, ok := lru.data[key]
	if !ok {
		return nil, false
	}
	item := v.Value.(*lruItem)
	lru.ll.Remove(v)
	delete(lru.data, key)
	return item.value, true
}
