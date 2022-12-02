package tinylfu

type slruItem[V any] struct {
	listid int
	key    string
	value  V
	keyh   uint64
}

// Cache is an LRU cache.  It is not safe for concurrent access.
type slruCache[V any] struct {
	data           map[string]*Element[slruItem[V]]
	onecap, twocap int
	one, two       *List[slruItem[V]]
}

func newSLRU[V any](onecap, twocap int, data map[string]*Element[slruItem[V]]) *slruCache[V] {
	return &slruCache[V]{
		data:   data,
		onecap: onecap,
		one:    NewList[slruItem[V]](),
		twocap: twocap,
		two:    NewList[slruItem[V]](),
	}
}

// xxget updates the cache data structures for a get
func (slru *slruCache[V]) get(v *Element[slruItem[V]]) {

	// already on list two?
	if v.Value.listid == 2 {
		slru.two.MoveToFront(v)
		return
	}

	// must be list one

	// is there space on the next list?
	if slru.two.Len() < slru.twocap {
		// just do the remove/add
		slru.one.Remove(v)
		v.Value.listid = 2
		slru.data[v.Value.key] = slru.two.PushFront(v.Value)
		return
	}

	back := slru.two.Back()

	// swap the key/values
	*back.Value, *v.Value = *v.Value, *back.Value

	back.Value.listid = 2
	v.Value.listid = 1

	// update pointers in the map
	slru.data[v.Value.key] = v
	slru.data[back.Value.key] = back

	// move the elements to the front of their lists
	slru.one.MoveToFront(v)
	slru.two.MoveToFront(back)
}

// Set sets a value in the cache
func (slru *slruCache[V]) add(newitem slruItem[V]) {

	newitem.listid = 1

	if slru.one.Len() < slru.onecap || (slru.Len() < slru.onecap+slru.twocap) {
		slru.data[newitem.key] = slru.one.PushFront(&newitem)
		return
	}

	// reuse the tail item
	e := slru.one.Back()

	delete(slru.data, e.Value.key)

	*e.Value = newitem

	slru.data[e.Value.key] = e
	slru.one.MoveToFront(e)
}

func (slru *slruCache[V]) victim() *Element[slruItem[V]] {

	if slru.Len() < slru.onecap+slru.twocap {
		return nil
	}

	v := slru.one.Back()

	return v
}

// Len returns the total number of items in the cache
func (slru *slruCache[V]) Len() int {
	return slru.one.Len() + slru.two.Len()
}

// Remove removes an item from the cache, returning the item and a boolean indicating if it was found
func (slru *slruCache[V]) Remove(key string) (*V, bool) {
	v, ok := slru.data[key]
	if !ok {
		return nil, false
	}

	if v.Value.listid == 2 {
		slru.two.Remove(v)
	} else {
		slru.one.Remove(v)
	}

	delete(slru.data, key)

	return &v.Value.value, true
}
