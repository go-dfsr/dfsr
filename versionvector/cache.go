package versionvector

import (
	"time"

	"gopkg.in/dfsr.v0/cache"

	"github.com/go-ole/go-ole"
)

// Lookup defines a version vector lookup function.
type Lookup func(group ole.GUID) (vector *Vector, err error)

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

// Set addes the vector to the cache for the given group. If a value already
// exists for that group it is replaced.
func (cache *Cache) Set(group ole.GUID, vector *Vector) {
	cache.c.Set(group, vector)
}

// Value returns the cached vector for the given group if it exists and has
// not expired. When a value is present ok will be true, otherwise it will be
// false.
func (cache *Cache) Value(group ole.GUID) (vector *Vector, ok bool) {
	v, ok := cache.c.Value(group)
	if ok {
		vector, _ = v.(*Vector).Duplicate()
	}
	return
}

// Lookup returns the cached vector for the given group if there is an unexpired
// value already present in the cache. If the cached value is missing or
// expired, a lookup will be performed and the result of that lookup will be
// returned.
//
// TODO: Add context after Go 1.7 is released?
func (cache *Cache) Lookup(group ole.GUID) (vector *Vector, err error) {
	v, err := cache.c.Lookup(group)
	return v.(*Vector).Duplicate()
}
