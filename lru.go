package main

import "fmt"

// Cache is an LRU cache.  It is not safe for concurrent access.
type lruCache[V any] struct {
	data map[string]*Element[slruItem[V]]
	cap  int
	ll   *List[slruItem[V]]
}

func newLRU[V any](cap int, data map[string]*Element[slruItem[V]]) *lruCache[V] {
	return &lruCache[V]{
		data: data,
		cap:  cap,
		ll:   NewList[slruItem[V]](),
	}
}

// Get returns a value from the cache
func (lru *lruCache[V]) get(v *Element[slruItem[V]]) {
	lru.ll.MoveToFront(v)
}

// Set sets a value in the cache
func (lru *lruCache[V]) add(newitem slruItem[V]) (oitem slruItem[V], evicted bool) {
	if lru.ll.Len() < lru.cap {
		lru.data[newitem.key] = lru.ll.PushFront(&newitem)
		return slruItem[V]{}, false
	}

	// reuse the tail item
	e := lru.ll.Back()
	if e == nil {
		en := lru.ll.Len()
		fmt.Println(en)
		fmt.Println("aryeh")
	}
	delete(lru.data, e.Value.key)

	oitem = *e.Value
	*e.Value = newitem

	lru.data[e.Value.key] = e
	lru.ll.MoveToFront(e)

	return oitem, true
}

// Len returns the total number of items in the cache
func (lru *lruCache[V]) Len() int {
	return len(lru.data)
}

// Remove removes an item from the cache, returning the item and a boolean indicating if it was found
func (lru *lruCache[V]) Remove(key string) (*V, bool) {
	v, ok := lru.data[key]
	if !ok {
		return nil, false
	}
	lru.ll.Remove(v)
	delete(lru.data, key)
	return &v.Value.value, true
}
