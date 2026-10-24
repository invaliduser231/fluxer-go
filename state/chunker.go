package state

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

// ErrChunkTimeout is returned by Chunker.Request when not all chunks arrived within
// Chunker.Timeout.
var ErrChunkTimeout = errors.New("state: member chunks did not arrive in time")

// MemberRequester sends a request guild members command. *fluxer.Session implements
// it.
type MemberRequester interface {
	RequestGuildMembers(ctx context.Context, r fluxer.RequestMembers) error
}

// RateLimitedError is returned by Chunker.Request when the gateway answers the
// request with RATE_LIMITED.
type RateLimitedError struct {
	GuildID string
	// RetryAfter is how long to wait before requesting members of the guild again.
	RetryAfter time.Duration
}

// Error implements the error interface.
func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("state: member request for %s is rate limited for %s", e.GuildID, e.RetryAfter)
}

// Chunker requests guild members over the gateway and collects the matching
// GUILD_MEMBERS_CHUNK events. Pass raw dispatches to Handle.
type Chunker struct {
	// Requester sends the member requests.
	Requester MemberRequester
	// Filter, when set, makes Handle request all members of each guild in a
	// GUILD_CREATE for which it returns true. ChunkAll and ChunkGuilds are ready-made
	// filters.
	Filter func(guildID string) bool
	// Timeout is how long Request waits for all chunks after sending the request. The
	// default is 30 seconds.
	Timeout time.Duration
	// OnError receives errors from the requests started by Filter.
	OnError func(guildID string, err error)

	mu      sync.Mutex
	pending map[string]*chunkRequest
}

type chunkRequest struct {
	guildID   string
	members   []fluxer.Member
	presences []fluxer.Presence
	seen      map[int]bool
	count     int
	err       error
	done      chan struct{}
	closed    bool
}

func (r *chunkRequest) finish() {
	if !r.closed {
		close(r.done)
		r.closed = true
	}
}

// NewChunker returns a chunker that sends requests through r.
func NewChunker(r MemberRequester) *Chunker {
	return &Chunker{Requester: r, pending: map[string]*chunkRequest{}}
}

// ChunkAll is a Chunker.Filter that matches every guild.
func ChunkAll(string) bool { return true }

// ChunkGuilds returns a Chunker.Filter that matches the listed guilds.
func ChunkGuilds(ids ...string) func(string) bool {
	return func(id string) bool { return slices.Contains(ids, id) }
}

func newNonce() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Request sends r with a new nonce, waits until every chunk for that nonce has
// arrived and returns the collected members. Presences in the chunks are not
// returned. It fails with a *RateLimitedError if the gateway rate limits the
// request, with ErrChunkTimeout after Timeout, or with the context's error.
func (k *Chunker) Request(ctx context.Context, r fluxer.RequestMembers) ([]fluxer.Member, error) {
	r.Nonce = newNonce()
	req := &chunkRequest{guildID: r.GuildID, seen: map[int]bool{}, done: make(chan struct{})}
	k.mu.Lock()
	if k.pending == nil {
		k.pending = map[string]*chunkRequest{}
	}
	k.pending[r.Nonce] = req
	k.mu.Unlock()
	defer func() {
		k.mu.Lock()
		delete(k.pending, r.Nonce)
		k.mu.Unlock()
	}()
	if err := k.Requester.RequestGuildMembers(ctx, r); err != nil {
		return nil, err
	}
	timer := time.NewTimer(cmp.Or(k.Timeout, 30*time.Second))
	defer timer.Stop()
	select {
	case <-req.done:
	case <-timer.C:
		return nil, ErrChunkTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if req.err != nil {
		return nil, req.err
	}
	return req.members, nil
}

// RequestAll requests every member of a guild.
func (k *Chunker) RequestAll(ctx context.Context, guildID string) ([]fluxer.Member, error) {
	return k.Request(ctx, fluxer.RequestMembers{GuildID: guildID})
}

// Handle processes a raw gateway dispatch. It reads GUILD_MEMBERS_CHUNK,
// RATE_LIMITED and GUILD_CREATE and ignores other events. Chunks without a nonce or
// with a nonce the chunker did not send are ignored.
func (k *Chunker) Handle(event string, data json.RawMessage) {
	switch event {
	case "GUILD_MEMBERS_CHUNK":
		var e fluxer.GuildMembersChunk
		if json.Unmarshal(data, &e) != nil || e.Nonce == "" {
			return
		}
		k.mu.Lock()
		defer k.mu.Unlock()
		req := k.pending[e.Nonce]
		if req == nil || req.seen[e.ChunkIndex] {
			return
		}
		req.seen[e.ChunkIndex] = true
		req.count = e.ChunkCount
		for _, m := range e.Members {
			m.GuildID = e.GuildID
			req.members = append(req.members, m)
		}
		req.presences = append(req.presences, e.Presences...)
		if len(req.seen) >= req.count {
			req.finish()
		}
	case "RATE_LIMITED":
		var e fluxer.RateLimited
		if json.Unmarshal(data, &e) != nil || e.Opcode != 8 {
			return
		}
		k.mu.Lock()
		defer k.mu.Unlock()
		for nonce, req := range k.pending {
			if nonce == e.Meta["nonce"] || e.Meta["nonce"] == "" && req.guildID == e.Meta["guild_id"] {
				req.err = &RateLimitedError{GuildID: req.guildID, RetryAfter: time.Duration(e.RetryAfter * float64(time.Second))}
				req.finish()
				delete(k.pending, nonce)
			}
		}
	case "GUILD_CREATE":
		if k.Filter == nil {
			return
		}
		var e struct {
			ID          string `json:"id"`
			Unavailable bool   `json:"unavailable"`
		}
		if json.Unmarshal(data, &e) != nil || e.Unavailable || !k.Filter(e.ID) {
			return
		}
		go func() {
			if _, err := k.RequestAll(context.Background(), e.ID); err != nil && k.OnError != nil {
				k.OnError(e.ID, err)
			}
		}()
	}
}

type readyState struct {
	mu      sync.Mutex
	pending map[string]bool
	done    chan struct{}
	closed  bool
}

func (r *readyState) start(guilds []fluxer.UnavailableGuild) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pending = map[string]bool{}
	for _, g := range guilds {
		r.pending[g.ID] = true
	}
	if r.done == nil || r.closed {
		r.done, r.closed = make(chan struct{}), false
	}
	if len(r.pending) == 0 {
		r.finish()
	}
}

func (r *readyState) finish() {
	if !r.closed {
		close(r.done)
		r.closed = true
	}
}

func (r *readyState) resolve(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pending == nil || !r.pending[id] {
		return
	}
	delete(r.pending, id)
	if len(r.pending) == 0 {
		r.finish()
	}
}

func (r *readyState) wait() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done == nil {
		r.done = make(chan struct{})
	}
	return r.done
}

// WaitGuilds blocks until every guild listed in the last READY has arrived through
// GUILD_CREATE or GUILD_DELETE, or until ctx is done.
func (c *Cache) WaitGuilds(ctx context.Context) error {
	select {
	case <-c.ready.wait():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PendingGuilds returns how many guilds from the last READY have not arrived yet.
func (c *Cache) PendingGuilds() int {
	c.ready.mu.Lock()
	defer c.ready.mu.Unlock()
	return len(c.ready.pending)
}
