package tinylfu

import (
	"hash/maphash"
	"testing"
)

func TestAddAlreadyInCache(t *testing.T) {
	s := maphash.MakeSeed()
	c := New[string, string](100, 10000,
		func(k string) uint64 { return maphash.String(s, k) },
		func(k, v string) {},
	)

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

var SinkString string
var SinkBool bool

func BenchmarkGet(b *testing.B) {
	s := maphash.MakeSeed()
	t := New[string, string](64, 640,
		func(k string) uint64 { return maphash.String(s, k) },
		func(k, v string) {},
	)
	key := "some arbitrary key"
	val := "some arbitrary value"
	t.Add(key, val)
	for i := 0; i < b.N; i++ {
		SinkString, SinkBool = t.Get(key)
	}
}
