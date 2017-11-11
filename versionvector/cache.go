package versionvector

import (
	"context"
	"time"

	"gopkg.in/dfsr.v0/cache"
	"gopkg.in/dfsr.v0/callstat"
	"gopkg.in/dfsr.v0/dfsr"

	"github.com/go-ole/go-ole"
)

// Lookup defines a version vector lookup function.
type Lookup func(ctx context.Context, guid ole.GUID, tracker dfsr.Tracker) (vector *Vector, call callstat.Call, err error)

type entry struct {
	vector *Vector
	call   callstat.Call
}

func castLookup(lookup Lookup) cache.Lookup {
	return func(ctx context.Context, key cache.Key, tracker dfsr.Tracker) (value cache.Value, err error) {
		vector, call, err := lookup(ctx, key.(ole.GUID), tracker)
		value = entry{
			vector: vector,
			call:   call,
		}
		return
	}
}

// Cache is a threadsafe expiring cache of version vectors that is capable of
// passing through cache misses to a lookup function.
type Cache struct {
	c      *cache.Cache
	lookup Lookup
}

// NewCache returns a new version vector cache with the given cache duration and
// value lookup function.
func NewCache(duration time.Duration, lookup Lookup) *Cache {
	return &Cache{
		c:      cache.New(duration, castLookup(lookup)),
		lookup: lookup,
	}
}

// Close will release any resources consumed by the cache and its contents. It
// will also prevent further use of the cache.
func (cache *Cache) Close() {
	cache.c.Close()
}

// Evict will expuge all existing values from the cache. Outstanding lookups
// that are still pending will not be affected.
func (cache *Cache) Evict() {
	cache.c.Evict()
}

// Set adds the vector to the cache for the given GUID. If a value already
// exists in the cache for that GUID, the existing value is replaced.
//
// If the cache has been closed then Set will do nothing.
func (cache *Cache) Set(guid ole.GUID, vector *Vector, call callstat.Call) {
	cache.c.Set(guid, entry{
		vector: vector,
		call:   call,
	})
}

// Value returns the cached vector for the given GUID if it exists in the cache
// and has not expired. If the cached value is missing or expired, ok will be
// false.
//
// If the cache has been closed then ok will be false.
func (cache *Cache) Value(guid ole.GUID) (vector *Vector, call callstat.Call, ok bool) {
	v, ok := cache.c.Value(guid)
	if ok {
		e := v.(entry)
		var err error
		vector, err = e.vector.Duplicate()
		if err != nil {
			ok = false
			return
		}
		call = e.call
	}
	return
}

// Lookup returns the cached vector for the given GUID if it exists in the
// cache and has not expired. If the cached value is missing or expired, a
// lookup will be performed.
//
// If the cache has been closed then ErrClosed will be returned.
func (cache *Cache) Lookup(ctx context.Context, guid ole.GUID, tracker dfsr.Tracker) (vector *Vector, call callstat.Call, err error) {
	call.Begin("Cache.Lookup")
	defer call.Complete(err)

	v, err := cache.c.Lookup(ctx, guid, tracker)
	if err != nil {
		// Values aren't cached when an error comes back, so it's safe to return
		// the unduplicated value here. In all likelihood v should be nil here
		// anwyay.
		if v == nil {
			return
		}
		e := v.(entry)
		call.Add(&e.call)
		vector = e.vector
		return
	}
	e := v.(entry)
	call.Add(&e.call)
	vector, err = e.vector.Duplicate()
	return
}
