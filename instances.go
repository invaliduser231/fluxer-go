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
	flight map[string]*discovery
	pacers map[string]*Pacer
}

type discovery struct {
	done chan struct{}
	inst *Instance
	err  error
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
// discovery succeeds again. Concurrent calls for the same origin share one
// discovery, which keeps running when the caller that started it gives up, so
// ctx only bounds how long Get waits.
func (r *Instances) Get(ctx context.Context, origin string) (*Instance, error) {
	origin, err := NormalizeOrigin(origin)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	c, ok := r.cache[origin]
	if ok && time.Now().Before(c.until) {
		r.mu.Unlock()
		if c.inst != nil {
			return c.inst, nil
		}
		return nil, c.err
	}
	d := r.flight[origin]
	if d == nil {
		d = &discovery{done: make(chan struct{})}
		if r.flight == nil {
			r.flight = map[string]*discovery{}
		}
		r.flight[origin] = d
		go r.discover(context.WithoutCancel(ctx), origin, d)
	}
	r.mu.Unlock()
	select {
	case <-d.done:
		return d.inst, d.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *Instances) discover(ctx context.Context, origin string, d *discovery) {
	inst, err := DiscoverWith(ctx, cmp.Or(r.HTTP, http.DefaultClient), origin)
	r.mu.Lock()
	defer close(d.done)
	defer r.mu.Unlock()
	delete(r.flight, origin)
	if r.cache == nil {
		r.cache = map[string]cachedInstance{}
	}
	if err != nil {
		stale := r.cache[origin].inst
		r.cache[origin] = cachedInstance{inst: stale, err: err, until: time.Now().Add(cmp.Or(r.FailureTTL, time.Minute))}
		d.inst, d.err = stale, err
		if stale != nil {
			d.err = nil
		}
		return
	}
	r.cache[origin] = cachedInstance{inst: inst, until: time.Now().Add(cmp.Or(r.TTL, time.Hour))}
	d.inst = inst
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
// creates it on first use. origin is normalized like in Get; when that fails,
// it is used as given.
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
