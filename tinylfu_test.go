package tinylfu

import (
	"hash/maphash"
	"slices"
	"testing"
)

func TestAddAlreadyInCache(t *testing.T) {
	s := maphash.MakeSeed()
	c := New[string, string](100, 10000, func(k string) uint64 {
		return maphash.String(s, k)
	})

	c.Add("foo", "bar")

	val, _ := c.Get("foo")
	if val != "bar" {
		t.Errorf("c.Get(foo)=%q, want %q", val, "bar")
	}

	c.Add("foo", "baz")

	val, _ = c.Get("foo")
	if val != "baz" {
		t.Errorf("c.Get(foo)=%q, want %q", val, "baz")
	}
}

func TestOnEvict(t *testing.T) {
	type item struct {
		k, v string
	}

	var evicted []item
	var expected = []item{
		{k: "A", v: "1"},
		{k: "B", v: "2"},
	}

	s := maphash.MakeSeed()
	c := New[string, string](64, 640,
		func(k string) uint64 {
			return maphash.String(s, k)
		},
		OnEvict(func(k, v string) {
			evicted = append(evicted, item{k, v})
		}),
	)

	c.Add("A", "1")
	c.Add("B", "2")
	c.Add("C", "3")

	if !slices.Equal(evicted, expected) {
		t.Errorf("evicted=%+v, expected=%+v", evicted, expected)
	}
}

func TestOnReplace(t *testing.T) {
	type item struct {
		k, v string
	}

	var evicted []item
	var expectedEvicted = []item{
		{k: "A", v: "1"},
	}

	var replaced []item
	var expectedReplaced = []item{
		{k: "A", v: "1"},
	}

	s := maphash.MakeSeed()
	c := New[string, string](64, 640,
		func(k string) uint64 {
			return maphash.String(s, k)
		},
		OnEvict(func(k, v string) {
			evicted = append(evicted, item{k, v})
		}),
		OnReplace(func(k, v string) {
			replaced = append(replaced, item{k, v})
		}),
	)

	c.Add("A", "1")
	c.Add("B", "2")
	c.Add("A", "3")

	if !slices.Equal(evicted, expectedEvicted) {
		t.Errorf("evicted=%+v, expected=%+v", evicted, expectedEvicted)
	}
	if !slices.Equal(replaced, expectedReplaced) {
		t.Errorf("replaced=%+v, expected=%+v", replaced, expectedReplaced)
	}
}

var SinkString string
var SinkBool bool

func BenchmarkGet(b *testing.B) {
	s := maphash.MakeSeed()
	t := New[string, string](64, 640, func(k string) uint64 {
		return maphash.String(s, k)
	})
	key := "some arbitrary key"
	val := "some arbitrary value"
	t.Add(key, val)
	for i := 0; i < b.N; i++ {
		SinkString, SinkBool = t.Get(key)
	}
}
