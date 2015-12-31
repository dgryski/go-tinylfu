package tinylfu

import "container/list"

type slruItem struct {
	second bool
	key    string
	value  interface{}
	keyh   uint64
}

// Cache is an LRU cache.  It is not safe for concurrent access.
type slruCache struct {
	data           map[string]*list.Element
	onecap, twocap int
	one, two       *list.List
}

func newSLRU(onecap, twocap int) *slruCache {
	return &slruCache{
		data:   make(map[string]*list.Element),
		onecap: onecap,
		one:    list.New(),
		twocap: twocap,
		two:    list.New(),
	}
}

// Get returns a value from the cache
func (slru *slruCache) Get(key string) (interface{}, bool) {
	v, ok := slru.data[key]

	if !ok {
		return nil, false
	}

	item := v.Value.(*slruItem)

	// already on list two?
	if item.second {
		slru.two.MoveToFront(v)
		return item.value, true
	}

	// must be list one

	// is there space on the next list?
	if slru.two.Len() < slru.twocap {
		// just do the remove/add
		slru.one.Remove(v)
		item.second = true
		slru.data[key] = slru.two.PushFront(item)
		return item.value, true
	}

	back := slru.two.Back()
	bitem := back.Value.(*slruItem)

	// swap the key/values
	bitem.key, item.key = item.key, bitem.key
	bitem.value, item.value = item.value, bitem.value
	bitem.keyh, item.keyh = item.keyh, bitem.keyh

	// update pointers in the map
	slru.data[item.key] = v
	slru.data[bitem.key] = back

	// move the elements to the front of their lists
	slru.one.MoveToFront(v)
	slru.two.MoveToFront(back)

	return bitem.value, true
}

// Set sets a value in the cache
func (slru *slruCache) Add(key string, value interface{}, keyh uint64) {
	if slru.one.Len() < slru.onecap {
		slru.data[key] = slru.one.PushFront(&slruItem{false, key, value, keyh})
		return
	}

	// reuse the tail item
	e := slru.one.Back()
	item := e.Value.(*slruItem)

	delete(slru.data, item.key)
	item.key = key
	item.value = value
	item.keyh = keyh
	slru.data[key] = e
	slru.one.MoveToFront(e)
}

// Len returns the total number of items in the cache
func (slru *slruCache) Len() int {
	return len(slru.data)
}

// Remove removes an item from the cache, returning the item and a boolean indicating if it was found
func (slru *slruCache) Remove(key string) (interface{}, bool) {
	v, ok := slru.data[key]
	if !ok {
		return nil, false
	}

	item := v.Value.(*slruItem)

	if item.second {
		slru.two.Remove(v)
	} else {
		slru.one.Remove(v)
	}

	delete(slru.data, key)

	return item.value, true
}
