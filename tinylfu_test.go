package tinylfu

import "testing"

func TestAddAlreadyInCache(t *testing.T) {
	c := New(100, 10000)

	c.Add("foo", "bar")

	val, _ := c.Get("foo")
	if val.(string) != "bar" {
		t.Errorf("c.Get(foo)=%q, want %q", val, "bar")
	}

	c.Add("foo", "baz")

	val, _ = c.Get("foo")
	if val.(string) != "baz" {
		t.Errorf("c.Get(foo)=%q, want %q", val, "baz")
	}
}
