package tinylfu

import (
	"testing"
)

func TestNvec(t *testing.T) {

	n := newNvec(8)

	n.inc(0)
	if n[0] != 0x01 {
		t.Errorf("n[0]=%x, want 0x10: (n=% 02x)", n[0], n)
	}
	if w := n.get(0); w != 1 {
		t.Errorf("n.get(0)=%x, want 1", w)
	}
	if w := n.get(1); w != 0 {
		t.Errorf("n.get(1)=%x, want 0", w)
	}

	n.inc(1)
	if n[0] != 0x11 {
		t.Errorf("n[0]=%x, want 0x11: (n=% 02x)", n[0], n)
	}
	if w := n.get(0); w != 1 {
		t.Errorf("n.get(0)=%x, want 1", w)
	}
	if w := n.get(1); w != 1 {
		t.Errorf("n.get(1)=%x, want 1", w)
	}

	for i := 0; i < 14; i++ {
		n.inc(1)
	}
	if n[0] != 0xf1 {
		t.Errorf("n[1]=%x, want 0x1f: (n=% 02x)", n[0], n)
	}
	if w := n.get(1); w != 15 {
		t.Errorf("n.get(1)=%x, want 15", w)
	}
	if w := n.get(0); w != 1 {
		t.Errorf("n.get(0)=%x, want 1", w)
	}

	// ensure clamped
	for i := 0; i < 3; i++ {
		n.inc(1)
		if n[0] != 0xf1 {
			t.Errorf("n[0]=%x, want 0x1f: (n=% 02x)", n[0], n)
		}
	}

	n.reset()

	if n[0] != 0x70 {
		t.Errorf("n[0]=%x, want 0x70 (n=% 02x)", n[0], n)
	}
}

func TestCM4(t *testing.T) {

	cm := newCM4(32)

	hash := uint64(0x0ddc0ffeebadf00d)

	cm.add(hash)
	cm.add(hash)

	if got := cm.estimate(hash); got != 2 {
		t.Errorf("cm.estimate(%x)=%d, want 2\n", hash, got)
	}
}
