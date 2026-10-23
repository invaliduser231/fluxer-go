package fluxer

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"sync"
	"time"
)

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

func NewInstances() *Instances { return &Instances{} }

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

func (r *Instances) Forget(origin string) {
	origin, err := NormalizeOrigin(origin)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cache, origin)
}

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

func (r *Instances) Pacer(origin string) *Pacer {
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

func (r *Instances) Client(ctx context.Context, origin, token string) (*Client, error) {
	inst, err := r.Get(ctx, origin)
	if err != nil {
		return nil, err
	}
	return inst.NewClient(token), nil
}

func (r *Instances) Session(ctx context.Context, origin, token string) (*Session, error) {
	inst, err := r.Get(ctx, origin)
	if err != nil {
		return nil, err
	}
	s := inst.NewSession(token)
	s.Pacer = r.Pacer(inst.Origin)
	return s, nil
}

func (i *Instance) NewSession(token string) *Session {
	return &Session{URL: i.Gateway, Token: token, HTTP: i.HTTP}
}
