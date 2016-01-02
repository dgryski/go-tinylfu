package tinylfu

import "container/list"

type slruItem struct {
	listid int
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

func newSLRU(onecap, twocap int, data map[string]*list.Element) *slruCache {
	return &slruCache{
		data:   data,
		onecap: onecap,
		one:    list.New(),
		twocap: twocap,
		two:    list.New(),
	}
}

// get updates the cache data structures for a get
func (slru *slruCache) get(v *list.Element) {
	item := v.Value.(*slruItem)

	// already on list two?
	if item.listid == 2 {
		slru.two.MoveToFront(v)
		return
	}

	// must be list one

	// is there space on the next list?
	if slru.two.Len() < slru.twocap {
		// just do the remove/add
		slru.one.Remove(v)
		item.listid = 2
		slru.data[item.key] = slru.two.PushFront(item)
		return
	}

	back := slru.two.Back()
	bitem := back.Value.(*slruItem)

	// swap the key/values
	*bitem, *item = *item, *bitem

	bitem.listid = 2
	item.listid = 1

	// update pointers in the map
	slru.data[item.key] = v
	slru.data[bitem.key] = back

	// move the elements to the front of their lists
	slru.one.MoveToFront(v)
	slru.two.MoveToFront(back)
}

// Set sets a value in the cache
func (slru *slruCache) add(newitem slruItem) {

	newitem.listid = 1

	if slru.one.Len() < slru.onecap {
		slru.data[newitem.key] = slru.one.PushFront(&newitem)
		return
	}

	// reuse the tail item
	e := slru.one.Back()
	item := e.Value.(*slruItem)

	delete(slru.data, item.key)

	*item = newitem

	slru.data[item.key] = e
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

	if item.listid == 2 {
		slru.two.Remove(v)
	} else {
		slru.one.Remove(v)
	}

	delete(slru.data, key)

	return item.value, true
}
