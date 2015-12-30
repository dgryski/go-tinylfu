// Package tinylfu is an implementation of the TinyLFU caching algorithm
/*
   http://arxiv.org/abs/1512.00727
*/
package tinylfu

import (
	"github.com/dchest/siphash"
)

type T struct {
	c       *cm4
	bouncer *doorkeeper
	w       int
	samples int
	lru     *lruCache
	slru    *slruCache
}

func New(size int, samples int) *T {

	lruSize := size / 100
	slruSize := int(float64(size) * (99.0 / 100.0))
	slru20 := int(0.2 * float64(slruSize))

	return &T{
		c:       newCM4(size),
		w:       0,
		samples: samples,
		bouncer: newDoorkeeper(samples, 0.01),

		lru:  newLRU(lruSize),
		slru: newSLRU(slru20, slruSize-slru20),
	}
}

func (t *T) Get(key string) (interface{}, bool) {

	t.w++
	if t.w == t.samples {
		t.c.reset()
		t.bouncer.reset()
		t.w = 0
	}

	keyh := siphash.Hash(0, 0, stringToSlice(key))

	t.c.add(keyh)

	if val, ok := t.lru.Get(key); ok {
		return val, ok
	}

	if val, ok := t.slru.Get(key); ok {
		return val, ok
	}

	return nil, false
}

func (t *T) Add(key string, val interface{}) {
	okey, oval, evicted := t.lru.Add(key, val)
	if !evicted {
		return
	}

	okeyh := siphash.Hash(0, 0, stringToSlice(okey))

	if !t.bouncer.allow(okeyh) {
		return
	}

	// estimate count of what will be evicted from slru
	// TODO(dgryski): find a way to do this without poking into slru internals
	victim := t.slru.one.Back()
	if victim == nil {
		t.slru.Add(okey, oval, okeyh)
		return
	}

	item := victim.Value.(*slruItem)
	vcount := t.c.estimate(item.keyh)
	ocount := t.c.estimate(okeyh)

	if ocount < vcount {
		return
	}

	t.slru.Add(okey, oval, okeyh)
}
