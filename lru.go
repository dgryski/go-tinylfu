package tinylfu

import "container/list"

// Cache is an LRU cache.  It is not safe for concurrent access.
type lruCache struct {
	data map[string]*list.Element
	cap  int
	ll   *list.List
}

func newLRU(cap int, data map[string]*list.Element) *lruCache {
	return &lruCache{
		data: data,
		cap:  cap,
		ll:   list.New(),
	}
}

// Get returns a value from the cache
func (lru *lruCache) get(v *list.Element) {
	lru.ll.MoveToFront(v)
}

// Set sets a value in the cache
func (lru *lruCache) add(newitem slruItem) (oitem slruItem, evicted bool) {
	if lru.ll.Len() < lru.cap {
		lru.data[newitem.key] = lru.ll.PushFront(&newitem)
		return slruItem{}, false
	}

	// reuse the tail item
	e := lru.ll.Back()
	item := e.Value.(*slruItem)

	delete(lru.data, item.key)

	oitem = *item
	*item = newitem

	lru.data[item.key] = e
	lru.ll.MoveToFront(e)

	return oitem, true
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
	item := v.Value.(*slruItem)
	lru.ll.Remove(v)
	delete(lru.data, key)
	return item.value, true
}
