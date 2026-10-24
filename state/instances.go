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
	Options     func(origin string) []Option

	mu     sync.Mutex
	opts   []Option
	caches map[string]*Cache
}

func NewInstances(opts ...Option) *Instances {
	return &Instances{opts: opts, caches: map[string]*Cache{}}
}

func (m *Instances) For(origin string) *Cache {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.caches == nil {
		m.caches = map[string]*Cache{}
	}
	c := m.caches[origin]
	if c == nil {
		opts := m.opts
		if m.Options != nil {
			opts = append(slices.Clone(opts), m.Options(origin)...)
		}
		c = New(opts...)
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
