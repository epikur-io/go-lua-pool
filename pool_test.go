package pool

import (
	"testing"
	"time"

	lua "github.com/epikur-io/go-lua"
)

func TestAqcuireAndRelease(t *testing.T) {
	lpool := NewPool(2)
	lvms := []*lua.State{}
	for range 2 {
		lvm := lpool.Acquire()
		lvms = append(lvms, lvm)
	}
	plen := lpool.Len()
	if plen != 0 {
		t.Errorf("pool expected to be empty but %d instances remained", plen)
	}

	// should timeout since pool is now empty
	_, err := lpool.AcquireTimeout(1 * time.Second)
	if err == nil {
		t.Errorf("expected timout error but got %v", err)
	}

	for _, lvm := range lvms {
		lpool.Release(lvm)
	}

	plen = lpool.Len()
	if plen != 2 {
		t.Errorf("pool expected to be full but got %d instances", plen)
	}
}
