// Package tinylfu is an implementation of the TinyLFU caching algorithm
/*
   http://arxiv.org/abs/1512.00727
*/
package tinylfu

import (
	"github.com/dgryski/go-metro"
)

type T[V any] struct {
	c              *cm4
	bouncer        *doorkeeper
	w              int
	samples        int
	lru            *lruCache[V]
	slru           *slruCache[V]
	data           map[string]*Element[slruItem[V]]
	hits           uint64
	misses         uint64
	lruPct         float32
	interval       int
	step           int
	percentage     float32
	wentUp         bool
	madeStepBigger bool
	lastSuccess    float32
	size           int
	withResize     bool
}

func New[V any](size int, samples int, withResize bool) *T[V] {

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
		step:       50000,

		withResize: withResize,
	}
}

func (t *T[V]) resize() {
	t.interval++

	if t.interval == t.step {

		success := float32(t.hits) / (float32(t.misses) + float32(t.hits))
		var newPct = t.lruPct
		if success >= t.lastSuccess {
			if t.wentUp {
				newPct = t.lruPct + t.percentage
				t.wentUp = true
			} else {
				newPct = t.lruPct - t.percentage
				t.wentUp = false
			}
		} else {
			if t.wentUp {
				newPct = (t.lruPct) - t.percentage
				t.wentUp = false
			} else {
				newPct = t.lruPct + t.percentage
				t.wentUp = true
			}
		}
		if newPct < 1 {
			newPct = 1
		}

		if newPct > 99 {
			newPct = 99
		}

		t.lruPct = newPct

		t.setCaps(float32(int(newPct)))
		t.percentage *= 0.98

		if t.lastSuccess-success < -0.05 || t.lastSuccess-success > 0.05 {
			t.percentage = 6.25
		}

		t.hits = 0
		t.misses = 0
		t.interval = 0
		t.lastSuccess = success
		t.resize1()
	}
}

func (t *T[V]) Get(key string) (*V, bool) {
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

const resize_gag = 1000

func (t *T[V]) resize1() {
	var resizes = 0
	for t.lru.ll.Len() > t.lru.cap && resizes < resize_gag {
		// reuse the tail item
		last := t.lru.ll.Back()
		t.lru.ll.Remove(last)
		t.slru.add(*last.Value)
		resizes++
	}

	for t.slru.one.Len() > (t.slru.onecap) && resizes < 1000 {
		// reuse the tail item
		last := t.slru.victim()
		if last == nil {
			break
		}
		if last.Value.listid == 2 {
			t.slru.two.Remove(last)
		} else {
			t.slru.one.Remove(last)
		}
		resizes++
		t.lru.add(*last.Value)
	}

	for t.slru.two.Len() > t.slru.twocap && resizes < 1000 {
		last := t.slru.two.Back()
		if last == nil {
			break
		}
		t.slru.two.Remove(last)

		resizes++
		t.lru.add(*last.Value)
	}
}

func (t *T[V]) setCaps(percentage float32) {
	if percentage < 0 {
		percentage = 0
	}

	if percentage > 100 {
		percentage = 100
	}
	t.lru.cap = int((percentage * float32(t.size)) / 100)
	if t.lru.cap < 1 {
		t.lru.cap = 1
	}

	slruSize := int(float32(t.size) * ((100.0 - percentage) / 100.0))

	slru20 := int(0.2 * float32(slruSize))

	t.slru.onecap = slru20
	t.slru.twocap = slruSize - slru20
	if t.slru.twocap < 1 {
		t.slru.twocap = 1
	}
	if t.slru.onecap < 1 {
		t.slru.onecap = 1
	}
}

func (t *T[V]) AllCaps() int {

	return t.lru.cap + t.slru.twocap + t.slru.onecap
}

func (t *T[V]) AllSizes() int {
	return t.slru.Len() + t.lru.ll.Len()
}
func (t *T[V]) AllKeys() int {
	return len(t.data)
}

func (t *T[V]) Add(key string, val V) {
	if t.withResize {
		t.resize()
	}
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

	vcount := t.c.estimate(victim.Value.keyh)
	ocount := t.c.estimate(oitem.keyh)

	if ocount < vcount {
		return
	}

	t.slru.add(oitem)
}
