// Package state keeps the guilds, channels, roles, members, messages and voice
// states a bot sees, built from gateway events.
//
// Feed a [Cache] from the session handler before the router, so handlers read
// the state after the event:
//
//	cache := state.New(state.WithMemberPolicy(state.MembersInVoice()))
//	session.Handler = func(event string, data json.RawMessage) {
//		cache.Handle(event, data)
//		router.Handle(event, data)
//	}
//
// Flags and policies decide what is kept. [OnChange] passes the value from
// before an update or delete. A [Chunker] loads guild members, a [Resolver]
// falls back to REST for anything missing, and a [Store] decides where the data
// lives. The default store is in memory; the state/redis module shares one
// cache between processes.
package state
