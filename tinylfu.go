// Package tinylfu is an implementation of the TinyLFU caching algorithm
/*
   http://arxiv.org/abs/1512.00727
*/
package tinylfu

import "github.com/dgryski/go-tinylfu/internal/list"

type T[K comparable, V any] struct {
	c       *cm4
	bouncer *doorkeeper
	w       int
	samples int
	lru     *lruCache[K, V]
	slru    *slruCache[K, V]
	data    map[K]*list.Element[*slruItem[K, V]]
	hash    func(K) uint64
	evict   func(K, V)
	replace func(K, V)
}

type Option[K comparable, V any] func(*T[K, V])

func OnEvict[K comparable, V any](f func(key K, old V)) Option[K, V] {
	return func(t *T[K, V]) { t.evict = f }
}

func OnReplace[K comparable, V any](f func(key K, old V)) Option[K, V] {
	return func(t *T[K, V]) { t.replace = f }
}

func New[K comparable, V any](size int, samples int, hash func(K) uint64, options ...Option[K, V]) *T[K, V] {
	const lruPct = 1

	lruSize := (lruPct * size) / 100
	if lruSize < 1 {
		lruSize = 1
	}
	slruSize := size - lruSize
	if slruSize < 1 {
		slruSize = 1
	}
	slru20 := slruSize / 5
	if slru20 < 1 {
		slru20 = 1
	}

	data := make(map[K]*list.Element[*slruItem[K, V]], size)

	t := &T[K, V]{
		c:       newCM4(size),
		w:       0,
		samples: samples,
		bouncer: newDoorkeeper(samples, 0.01),

		data: data,

		lru:  newLRU(lruSize, data),
		slru: newSLRU(slru20, slruSize-slru20, data),

		hash:    hash,
		evict:   ignore[K, V],
		replace: ignore[K, V],
	}

	for _, option := range options {
		option(t)
	}

	return t
}

func (t *T[K, V]) Get(key K) (V, bool) {

	t.w++
	if t.w == t.samples {
		t.c.reset()
		t.bouncer.reset()
		t.w = 0
	}

	val, ok := t.data[key]
	if !ok {
		keyh := t.hash(key)
		t.c.add(keyh)
		return *new(V), false
	}

	item := val.Value

	t.c.add(item.keyh)

	v := item.value
	if item.listid == 0 {
		t.lru.get(val)
	} else {
		t.slru.get(val)
	}

	return v, true
}

func (t *T[K, V]) Add(key K, val V) {

	if e, ok := t.data[key]; ok {
		// Key is already in our cache.
		// `Add` will act as a `Get` for list movements
		item := e.Value
		oval := item.value
		item.value = val
		t.c.add(item.keyh)

		if item.listid == 0 {
			t.lru.get(e)
		} else {
			t.slru.get(e)
		}

		t.replace(key, oval)
		return
	}

	newitem := slruItem[K, V]{0, key, val, t.hash(key)}

	oitem, evicted := t.lru.add(newitem)
	if !evicted {
		return
	}

	// estimate count of what will be evicted from slru
	victim := t.slru.victim()
	if victim == nil {
		if oitem, evicted := t.slru.add(oitem); evicted {
			t.evict(oitem.key, oitem.value)
		}
		return
	}

	if !t.bouncer.allow(oitem.keyh) {
		t.evict(oitem.key, oitem.value)
		return
	}

	vcount := t.c.estimate(victim.keyh)
	ocount := t.c.estimate(oitem.keyh)

	if ocount < vcount {
		t.evict(oitem.key, oitem.value)
		return
	}

	if oitem, evicted := t.slru.add(oitem); evicted {
		t.evict(oitem.key, oitem.value)
	}
}

func ignore[K, V any](K, V) {}
