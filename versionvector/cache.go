package versionvector

import (
	"time"

	"gopkg.in/dfsr.v0/cache"

	"github.com/go-ole/go-ole"
)

// Lookup defines a version vector lookup function.
type Lookup func(guid ole.GUID) (vector *Vector, err error)

func castLookup(lookup Lookup) cache.Lookup {
	return func(key cache.Key) (value cache.Value, err error) {
		return lookup(key.(ole.GUID))
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
func (cache *Cache) Set(guid ole.GUID, vector *Vector) {
	cache.c.Set(guid, vector)
}

// Value returns the cached vector for the given GUID if it exists in the cache
// and has not expired. If the cached value is missing or expired, ok will be
// false.
//
// If the cache has been closed then ok will be false.
func (cache *Cache) Value(guid ole.GUID) (vector *Vector, ok bool) {
	v, ok := cache.c.Value(guid)
	if ok {
		vector, _ = v.(*Vector).Duplicate()
	}
	return
}

// Lookup returns the cached vector for the given GUID if it exists in the
// cache and has not expired. If the cached value is missing or expired, a
// lookup will be performed.
//
// If the cache has been closed then ErrClosed will be returned.
//
// TODO: Add context after Go 1.7 is released?
func (cache *Cache) Lookup(guid ole.GUID) (vector *Vector, err error) {
	v, err := cache.c.Lookup(guid)
	if err != nil {
		// Values aren't cached when an error comes back, so it's safe to return
		// the unduplicated value here. In all likelihood v should be nil here
		// anwyay.
		return v.(*Vector), err
	}
	return v.(*Vector).Duplicate()
}
