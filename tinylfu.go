// Package tinylfu is an implementation of the TinyLFU caching algorithm
/*
   http://arxiv.org/abs/1512.00727
*/
package tinylfu

import (
	"container/list"
	"github.com/dgryski/go-metro"
)

type T struct {
	c       *cm4
	bouncer *doorkeeper
	w       int
	samples int
	lru     *lruCache
	slru    *slruCache
	data    map[string]*list.Element
}

func New(size int, samples int) *T {

	const lruPct = 1

	lruSize := (lruPct * size) / 100
	if lruSize < 1 {
		lruSize = 1
	}
	slruSize := int(float64(size) * ((100.0 - lruPct) / 100.0))
	if slruSize < 1 {
		slruSize  = 1

	}
	slru20 := int(0.2 * float64(slruSize))
	if slru20 < 1 {
		slru20 = 1
	}

	data := make(map[string]*list.Element, size)

	return &T{
		c:       newCM4(size),
		w:       0,
		samples: samples,
		bouncer: newDoorkeeper(samples, 0.01),

		data: data,

		lru:  newLRU(lruSize, data),
		slru: newSLRU(slru20, slruSize-slru20, data),
	}
}

func (t *T) Get(key string) (interface{}, bool) {

	t.w++
	if t.w == t.samples {
		t.c.reset()
		t.bouncer.reset()
		t.w = 0
	}

	val, ok := t.data[key]
	if !ok {
		keyh := metro.Hash64Str(key, 0)
		t.c.add(keyh)
		return nil, false
	}

	item := val.Value.(*slruItem)

	t.c.add(item.keyh)

	v := item.value
	if item.listid == 0 {
		t.lru.get(val)
	} else {
		t.slru.get(val)
	}

	return v, true
}

func (t *T) Add(key string, val interface{}) {

	newitem := slruItem{0, key, val, metro.Hash64Str(key, 0)}

	oitem, evicted := t.lru.add(newitem)
	if !evicted {
		return
	}

	// estimate count of what will be evicted from slru
	victim := t.slru.victim()
	if victim == nil {
		t.slru.add(oitem)
		return
	}

	if !t.bouncer.allow(oitem.keyh) {
		return
	}

	vcount := t.c.estimate(victim.keyh)
	ocount := t.c.estimate(oitem.keyh)

	if ocount < vcount {
		return
	}

	t.slru.add(oitem)
}
