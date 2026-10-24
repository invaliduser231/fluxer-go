package state

import (
	"encoding/json"
	"maps"
	"slices"
	"sync"
)

// Instances keeps one Cache per origin, for bots connected to several Fluxer
// instances. A cache is created the first time its origin is used.
type Instances struct {
	// SkipMembers is copied to each cache when it is created. Changing it later does not
	// affect existing caches.
	SkipMembers bool
	// OnError receives decode errors from all caches together with the origin that
	// produced them. It is read when a cache is created.
	OnError func(origin, event string, err error)
	// Options returns extra options for one origin. They are applied after the options
	// passed to NewInstances.
	Options func(origin string) []Option

	mu     sync.Mutex
	opts   []Option
	caches map[string]*Cache
}

// NewInstances returns an empty set of caches. opts apply to every cache it creates.
func NewInstances(opts ...Option) *Instances {
	return &Instances{opts: opts, caches: map[string]*Cache{}}
}

// For returns the cache for origin, creating it on the first call.
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

// Handler returns the Handle method of the cache for origin, for use as a raw
// gateway event handler.
func (m *Instances) Handler(origin string) func(event string, data json.RawMessage) {
	return m.For(origin).Handle
}

// Origins returns the origins that have a cache, sorted.
func (m *Instances) Origins() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Sorted(maps.Keys(m.caches))
}

// Drop forgets the cache for origin. The next call to For creates a new, empty one.
func (m *Instances) Drop(origin string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.caches, origin)
}
