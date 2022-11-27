// Package tinylfu is an implementation of the TinyLFU caching algorithm
/*
   http://arxiv.org/abs/1512.00727
*/
package tinylfu

import (
	"github.com/dgryski/go-metro"
)

type T[V any] struct {
	c           *cm4
	bouncer     *doorkeeper
	w           int
	samples     int
	lru         *lruCache[V]
	slru        *slruCache[V]
	data        map[string]*Element[slruItem[V]]
	hits        uint64
	misses      uint64
	lruPct      float32
	interval    int
	step        int
	percentage  float32
	wentUp      bool
	lastSuccess float32
	size        int
}

func New[V any](size int, samples int) *T[V] {

	const lruPct = 1
	lruSize := (lruPct * size) / 100
	if lruSize < 1 {
		lruSize = 1
	}
	slruSize := int(float64(size) * ((100.0 - lruPct) / 100.0))
	if slruSize < 1 {
		slruSize = 1

	}
	slru20 := int(0.2 * float64(slruSize))
	if slru20 < 1 {
		slru20 = 1
	}

	data := make(map[string]*Element[slruItem[V]], size)

	return &T[V]{
		c:       newCM4(size),
		w:       0,
		samples: samples,
		bouncer: newDoorkeeper(samples, 0.01),

		data: data,

		lru:  newLRU[V](lruSize, data),
		slru: newSLRU[V](slru20, slruSize-slru20, data),

		percentage: 6.25,
		size:       size,
		lruPct:     lruPct,
		step:       10,
	}
}

func (t *T[V]) Get(key string) (*V, bool) {
	t.interval++

	if t.interval == t.step {
		t.interval = 0

		success := float32(t.hits) / (float32(t.misses) + float32(t.hits))
		var newPct = t.lruPct
		if success >= t.lastSuccess {
			if t.wentUp {
				newPct = t.lruPct + t.percentage
			} else {
				newPct = t.lruPct - t.percentage
				t.wentUp = false
			}
		} else {
			if t.wentUp {
				newPct = (t.lruPct) - t.percentage
			} else {
				newPct = t.lruPct + t.percentage
				t.wentUp = true
			}
		}

		t.setCaps(newPct)

		t.percentage *= 0.98
		if t.lastSuccess-success < -0.05 || t.lastSuccess-success > 0.05 {
			t.percentage = 6.25

			if t.step < 5000000 {
				t.step *= 10
			}
		} else {
			if t.step > 100 {
				t.step /= 10
			}
		}

		t.lastSuccess = success
		t.hits = 0
		t.misses = 0
	}

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
		t.misses += 1
		return nil, false
	}

	t.hits += 1
	t.c.add(val.Value.keyh)

	v := val.Value.value
	if val.Value.listid == 0 {
		t.lru.get(val)
	} else {
		t.slru.get(val)
	}

	return &v, true
}

func (t *T[V]) setCaps(percentage float32) {
	if percentage < 0 {
		percentage = 0
	}

	if percentage > 100 {
		percentage = 100
	}
	t.lru.cap = (int(percentage) * t.size) / 100
	if t.lru.cap < 1 {
		t.lru.cap = 1
	}
	t.slru.twocap = int(float32(t.size) * ((100.0 - percentage) / 100.0))
	if t.slru.twocap < 1 {
		t.slru.twocap = 1
	}
	t.slru.onecap = int(0.2 * float64(t.slru.twocap))
	if t.slru.onecap < 1 {
		t.slru.onecap = 1
	}
}

func (t *T[V]) Add(key string, val V) {

	newitem := slruItem[V]{0, key, val, metro.Hash64Str(key, 0)}

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
