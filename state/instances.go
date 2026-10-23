package state

import (
	"encoding/json"
	"maps"
	"slices"
	"sync"
)

type Instances struct {
	SkipMembers bool
	OnError     func(origin, event string, err error)

	mu     sync.Mutex
	caches map[string]*Cache
}

func NewInstances() *Instances { return &Instances{caches: map[string]*Cache{}} }

func (m *Instances) For(origin string) *Cache {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.caches == nil {
		m.caches = map[string]*Cache{}
	}
	c := m.caches[origin]
	if c == nil {
		c = New()
		c.SkipMembers = m.SkipMembers
		if m.OnError != nil {
			c.OnError = func(event string, err error) { m.OnError(origin, event, err) }
		}
		m.caches[origin] = c
	}
	return c
}

func (m *Instances) Handler(origin string) func(event string, data json.RawMessage) {
	return m.For(origin).Handle
}

func (m *Instances) Origins() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Sorted(maps.Keys(m.caches))
}

func (m *Instances) Drop(origin string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.caches, origin)
}
