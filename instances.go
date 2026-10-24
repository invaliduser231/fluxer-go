package fluxer

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"sync"
	"time"
)

// Instances caches discovered instances by origin and hands out clients and
// gateway sessions for them. Successful lookups are cached for TTL, one hour by
// default, and failed ones for FailureTTL, one minute by default.
// IdentifiesPerMinute sets the gateway identify rate per instance and defaults
// to 60. The zero value is ready to use, and an Instances is safe for concurrent
// use.
type Instances struct {
	HTTP                *http.Client
	TTL                 time.Duration
	FailureTTL          time.Duration
	IdentifiesPerMinute int

	mu     sync.Mutex
	cache  map[string]cachedInstance
	pacers map[string]*Pacer
}

type cachedInstance struct {
	inst  *Instance
	err   error
	until time.Time
}

// NewInstances returns an empty Instances with default settings.
func NewInstances() *Instances { return &Instances{} }

// Get returns the instance at origin, running discovery unless a fresh cache
// entry exists. origin is normalized with NormalizeOrigin. When discovery fails
// and an earlier result is cached, Get keeps returning that result until
// discovery succeeds again. Failures caused by ctx ending are not cached.
// Concurrent calls for an uncached origin each run their own discovery.
func (r *Instances) Get(ctx context.Context, origin string) (*Instance, error) {
	origin, err := NormalizeOrigin(origin)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	c, ok := r.cache[origin]
	r.mu.Unlock()
	if ok && time.Now().Before(c.until) {
		if c.inst != nil {
			return c.inst, nil
		}
		return nil, c.err
	}
	inst, err := DiscoverWith(ctx, cmp.Or(r.HTTP, http.DefaultClient), origin)
	if err != nil && ctx.Err() != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cache == nil {
		r.cache = map[string]cachedInstance{}
	}
	if err != nil {
		stale := r.cache[origin].inst
		r.cache[origin] = cachedInstance{inst: stale, err: err, until: time.Now().Add(cmp.Or(r.FailureTTL, time.Minute))}
		if stale != nil {
			return stale, nil
		}
		return nil, err
	}
	r.cache[origin] = cachedInstance{inst: inst, until: time.Now().Add(cmp.Or(r.TTL, time.Hour))}
	return inst, nil
}

// Forget drops the cache entry for origin, so the next Get runs discovery.
func (r *Instances) Forget(origin string) {
	origin, err := NormalizeOrigin(origin)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cache, origin)
}

// Origins returns the sorted origins that have a cached instance, including
// entries past their TTL.
func (r *Instances) Origins() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.cache))
	for origin, c := range r.cache {
		if c.inst != nil {
			out = append(out, origin)
		}
	}
	slices.Sort(out)
	return out
}

// Pacer returns the identify pacer shared by all sessions of origin and
// creates it on first use. Unlike Get, it uses origin as given without
// normalizing it.
func (r *Instances) Pacer(origin string) *Pacer {
	if normalized, err := NormalizeOrigin(origin); err == nil {
		origin = normalized
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pacers == nil {
		r.pacers = map[string]*Pacer{}
	}
	p := r.pacers[origin]
	if p == nil {
		p = NewPacer(cmp.Or(r.IdentifiesPerMinute, 60))
		r.pacers[origin] = p
	}
	return p
}

// Client returns a Client for the instance at origin that authenticates with
// token.
func (r *Instances) Client(ctx context.Context, origin, token string) (*Client, error) {
	inst, err := r.Get(ctx, origin)
	if err != nil {
		return nil, err
	}
	return inst.NewClient(token), nil
}

// Session returns a gateway Session for the instance at origin. All sessions
// of one instance share the same identify Pacer.
func (r *Instances) Session(ctx context.Context, origin, token string) (*Session, error) {
	inst, err := r.Get(ctx, origin)
	if err != nil {
		return nil, err
	}
	s := inst.NewSession(token)
	s.Pacer = r.Pacer(inst.Origin)
	return s, nil
}

// NewSession returns a gateway Session for the instance's gateway URL, using
// the instance's HTTP client for the connection. Unlike NewClient, it does not
// strip a "Bot " prefix from token.
func (i *Instance) NewSession(token string) *Session {
	return &Session{URL: i.Gateway, Token: token, HTTP: i.HTTP}
}
