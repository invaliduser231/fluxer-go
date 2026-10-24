package fluxer

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
)

// Router decodes dispatches into typed events and calls the handlers registered for them. Its
// Handle method fits Session.Handler. The zero value is ready to use.
type Router struct {
	// OnError, when set, receives decode failures and recovered handler panics.
	OnError func(event string, err error)

	mu     sync.RWMutex
	routes map[string]*route
	any    []anyHandler
	nextID int
}

type route struct {
	decode   func(json.RawMessage) (any, error)
	handlers []typedHandler
}

type typedHandler struct {
	id int
	fn func(any)
}

type anyHandler struct {
	id int
	fn func(event string, data json.RawMessage)
}

// NewRouter returns an empty Router.
func NewRouter() *Router { return &Router{} }

// On registers fn for the event type P and returns a function that removes it. The data of each
// dispatch is decoded once and every handler of that type receives the same value. Empty or null
// data yields a zero value.
func On[T any, P interface {
	*T
	Event
}](r *Router, fn func(P)) func() {
	var zero T
	name := P(&zero).EventName()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.routes == nil {
		r.routes = map[string]*route{}
	}
	rt := r.routes[name]
	if rt == nil {
		rt = &route{decode: func(data json.RawMessage) (any, error) {
			v := P(new(T))
			if len(data) == 0 || string(data) == "null" {
				return v, nil
			}
			return v, json.Unmarshal(data, v)
		}}
		r.routes[name] = rt
	}
	r.nextID++
	id := r.nextID
	rt.handlers = append(rt.handlers, typedHandler{id: id, fn: func(v any) { fn(v.(P)) }})
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		rt.handlers = slices.DeleteFunc(rt.handlers, func(h typedHandler) bool { return h.id == id })
	}
}

// OnAny registers fn for every dispatch and returns a function that removes it. OnAny handlers run
// before the typed handlers.
func (r *Router) OnAny(fn func(event string, data json.RawMessage)) func() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	id := r.nextID
	r.any = append(r.any, anyHandler{id: id, fn: fn})
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.any = slices.DeleteFunc(r.any, func(h anyHandler) bool { return h.id == id })
	}
}

// Handle passes one dispatch to the OnAny handlers and then to the typed handlers for its name,
// each in registration order. Typed handlers are skipped when decoding fails. A panicking handler
// is recovered and reported to OnError, and the remaining handlers still run. Handlers added or
// removed during Handle take effect with the next dispatch.
func (r *Router) Handle(event string, data json.RawMessage) {
	r.mu.RLock()
	anys := slices.Clone(r.any)
	var decode func(json.RawMessage) (any, error)
	var handlers []typedHandler
	if rt := r.routes[event]; rt != nil {
		decode, handlers = rt.decode, slices.Clone(rt.handlers)
	}
	r.mu.RUnlock()

	for _, h := range anys {
		r.call(event, func() { h.fn(event, data) })
	}
	if len(handlers) == 0 {
		return
	}
	v, err := decode(data)
	if err != nil {
		r.fail(event, fmt.Errorf("fluxer: decoding %s: %w", event, err))
		return
	}
	for _, h := range handlers {
		r.call(event, func() { h.fn(v) })
	}
}

func (r *Router) call(event string, fn func()) {
	defer func() {
		if p := recover(); p != nil {
			r.fail(event, fmt.Errorf("fluxer: handler for %s panicked: %v", event, p))
		}
	}()
	fn()
}

func (r *Router) fail(event string, err error) {
	if r.OnError != nil {
		r.OnError(event, err)
	}
}
