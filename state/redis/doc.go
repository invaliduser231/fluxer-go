// Package redis stores a state.Cache in Redis, so several processes or shards
// share one cache.
//
//	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
//	store := redis.New(client, "mybot")
//	store.OnError = func(err error) { log.Print(err) }
//	cache := state.New(state.WithStore(store))
//
// Each table is kept as one hash per guild or channel under the prefix, with
// values encoded as JSON.
package redis
