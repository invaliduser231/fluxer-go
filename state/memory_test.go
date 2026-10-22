package state_test

import (
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/invaliduser231/fluxer-go/state"
)

func TestMemoryTable(t *testing.T) {
	tab := state.NewMemoryTable[int](true)
	tab.Put("g1", "a", 1)
	tab.Put("g1", "b", 2)
	tab.Put("g2", "c", 3)
	if v, ok := tab.Get("g1", "b"); !ok || v != 2 {
		t.Errorf("got %d, %v", v, ok)
	}
	if g, ok := tab.Locate("c"); !ok || g != "g2" {
		t.Errorf("locate got %q, %v", g, ok)
	}
	tab.Put("g3", "c", 4)
	tab.Delete("g2", "c")
	if g, ok := tab.Locate("c"); !ok || g != "g3" {
		t.Errorf("moved key located in %q, %v", g, ok)
	}
	for k := range tab.Group("g1") {
		tab.Delete("g1", k)
	}
	if tab.Len("g1") != 0 {
		t.Error("deleting while iterating failed")
	}
	groups := slices.Sorted(tab.Groups())
	if !slices.Equal(groups, []string{"g3"}) {
		t.Errorf("groups %v", groups)
	}
	tab.DeleteGroup("g3")
	if _, ok := tab.Locate("c"); ok {
		t.Error("index kept a deleted group")
	}
	plain := state.NewMemoryTable[int](false)
	plain.Put("g", "k", 1)
	if _, ok := plain.Locate("k"); ok {
		t.Error("unindexed table located a key")
	}
}

func TestMemoryTableConcurrent(t *testing.T) {
	tab := state.NewMemoryTable[int](true)
	var wg sync.WaitGroup
	for w := range 8 {
		wg.Go(func() {
			group := strconv.Itoa(w)
			for i := range 500 {
				key := group + "-" + strconv.Itoa(i)
				tab.Put(group, key, i)
				tab.Get(group, key)
				tab.Locate(key)
				if i%3 == 0 {
					tab.Delete(group, key)
				}
			}
		})
	}
	wg.Wait()
	total := 0
	for g := range tab.Groups() {
		total += tab.Len(g)
	}
	if total != 8*333 {
		t.Errorf("got %d entries", total)
	}
}
