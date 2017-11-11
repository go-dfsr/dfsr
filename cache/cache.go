// Package cache provides a generic threadsafe expiring cache implementation
// that is capable of passing cache misses through to a lookup function.
package cache

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"gopkg.in/dfsr.v0/dfsr"
)

const cacheSize = 32

// Key defines a cache key.
type Key interface{}

// Value defines a cache value.
type Value interface{}

// Releaser is an interface that cache values may implement to be released
// when their entries expire.
type Releaser interface {
	Release()
}

// Closer is an interface that cache values may implement to be closed
// when their entries expire.
type Closer interface {
	Close()
}

// Lookup defines a lookup function for retrieving new cache values.
type Lookup func(ctx context.Context, key Key, tracker dfsr.Tracker) (value Value, err error)

var (
	// ErrClosed is returned from calls to the cache or in the event that the
	// Close() function has already been called.
	ErrClosed = errors.New("The cache is closing or already closed.")
)

// Cache is a threadsafe expiring cache that is capable of passing cache misses
// through to a lookup function.
type Cache struct {
	d        time.Duration
	m        sync.RWMutex
	data     map[Key]*cacheEntry
	pending  map[Key]*pendingEntry
	log      []logEntry
	cleaning bool
	lookup   Lookup
}

type cacheEntry struct {
	Timestamp time.Time
	Key
	Value
}

type pendingEntry struct {
	WG sync.WaitGroup
	Value
	Err error
}

type logEntry struct {
	Timestamp time.Time
	Key
}

// New returns a new cache whose values survive for the given duration and are
// retrieved with the given lookup function.
func New(duration time.Duration, lookup Lookup) *Cache {
	return &Cache{
		d:       duration,
		data:    make(map[Key]*cacheEntry, cacheSize),
		pending: make(map[Key]*pendingEntry, cacheSize),
		log:     make([]logEntry, 0, cacheSize),
		lookup:  lookup,
	}
}

func (cache *Cache) closed() bool {
	return (cache.data == nil)
}

// Close will release any resources consumed by the cache and its contents. It
// will also prevent further use of the cache.
func (cache *Cache) Close() {
	cache.m.Lock()
	defer cache.m.Unlock()
	if cache.closed() {
		return
	}
	for _, entry := range cache.data {
		release(entry.Value)
	}
	cache.data = nil
	cache.pending = nil
	cache.log = nil
	cache.lookup = nil
}

// Evict will expuge all existing values from the cache. Outstanding lookups
// that are still pending will not be affected.
func (cache *Cache) Evict() {
	cache.m.Lock()
	defer cache.m.Unlock()
	if cache.closed() {
		return
	}
	for _, entry := range cache.data {
		release(entry.Value)
	}
	cache.data = make(map[Key]*cacheEntry, cacheSize)
	cache.log = make([]logEntry, 0, cacheSize)
}

// Set saves a value in the cache for the given key. If a value already exists
// in the cache for that key, the existing value is replaced.
//
// If the cache has been closed then Set will do nothing.
func (cache *Cache) Set(key Key, value Value) {
	now := time.Now()
	cache.m.Lock()
	defer cache.m.Unlock()
	if cache.closed() {
		return
	}
	cache.set(now, key, value)
}

// set does not acquire a lock. It is the caller's responsibility to maintain
// a read/write lock on the cache during the call.
func (cache *Cache) set(timestamp time.Time, k Key, v Value) {
	cache.delete(k)
	cache.data[k] = &cacheEntry{
		Timestamp: timestamp,
		Key:       k,
		Value:     v,
	}
	cache.log = append(cache.log, logEntry{Timestamp: timestamp, Key: k})
	cache.spawnCleanup()
}

// Value returns the value for the given key if it exists in the cache and has
// not expired. If the cached value is missing or expired, ok will be false.
//
// If the cache has been closed then ok will be false.
func (cache *Cache) Value(key Key) (value Value, ok bool) {
	cache.m.RLock()
	if cache.closed() {
		return
	}
	defer cache.m.RUnlock()
	return cache.value(key)
}

// value does not acquire a lock. It is the caller's responsibility to maintain
// a read lock on the cache during the call.
func (cache *Cache) value(k Key) (value Value, ok bool) {
	entry, found := cache.data[k]
	if found {
		expiration := entry.Timestamp.Add(cache.d)
		if expiration.After(time.Now()) {
			return entry.Value, true
		}
	}
	return nil, false
}

// Lookup returns the value for the given key if it exists in the cache and has
// not expired. If the cached value is missing or expired, a lookup will be
// performed.
//
// If the cache has been closed then ErrClosed will be returned.
func (cache *Cache) Lookup(ctx context.Context, key Key, tracker dfsr.Tracker) (value Value, err error) {
	// First attempt with read lock
	cache.m.RLock()
	if cache.closed() {
		return nil, ErrClosed
	}
	value, found := cache.value(key)
	cache.m.RUnlock()
	if found {
		return value, nil
	}

	// Second attempt with write lock
	cache.m.Lock()
	if cache.closed() {
		return nil, ErrClosed
	}
	value, found = cache.value(key)
	if found {
		cache.m.Unlock()
		return value, nil
	}

	// Wait for a response
	p := cache.pend(ctx, key, tracker)
	cache.m.Unlock()
	p.WG.Wait()

	return p.Value, p.Err
}

// pend does not acquire a lock. It is the caller's responsibility to maintain
// a read/write lock on the cache during the call.
//
// FIXME: Correctly handle contexts when there are multiple pending callers.
func (cache *Cache) pend(ctx context.Context, k Key, tracker dfsr.Tracker) (p *pendingEntry) {
	p, found := cache.pending[k]
	if !found {
		p = &pendingEntry{}
		cache.pending[k] = p
		p.WG.Add(1)
		go cache.retrieve(ctx, k, p, tracker) // FIXME: Create a composite context that only cancels if all of its members also cancel?
	}
	return
}

func (cache *Cache) retrieve(ctx context.Context, k Key, p *pendingEntry, tracker dfsr.Tracker) {
	// Handle cancellation
	select {
	case <-ctx.Done():
		p.Err = ctx.Err()
		return
	default:
	}

	p.Value, p.Err = cache.lookup(ctx, k, tracker) // This may block for some time
	now := time.Now()

	cache.m.Lock()
	if !cache.closed() {
		if p.Err == nil {
			cache.set(now, k, p.Value)
		}
		delete(cache.pending, k)
	}
	cache.m.Unlock()
	p.WG.Done()
}

// delete will remove the value with the given key from the cache if it exists.
// It releases any resources that were consumed by the value.
//
// delete does not acquire a lock. It is the caller's responsibility to
// maintain a read/write lock on the cache during the call.
func (cache *Cache) delete(k Key) {
	entry, found := cache.data[k]
	if found {
		release(entry.Value)
		delete(cache.data, k)
	}
}

// spawnCleanup will spawn a cleanup goroutine if it's needed and one isn't
// already running.
//
// spawnCleanup does not acquire a lock. It is the caller's responsibility to
// maintain a read/write lock on the cache during the call.
func (cache *Cache) spawnCleanup() {
	if len(cache.log) > 0 && !cache.cleaning {
		cache.cleaning = true
		go cache.cleanup(cache.log[0].Timestamp)
	}
}

// cleanup is run on its own goroutine and removes expired entries from the
// cache over time. It runs until the cache is empty, at which point it exits.
func (cache *Cache) cleanup(next time.Time) {
	var more bool
	for {
		now := time.Now()
		if next.After(now) {
			remaining := next.Sub(now)
			<-time.After(remaining)
			now = time.Now()
		}
		cache.m.Lock()
		next, more = cache.validate(now)
		if !more {
			cache.cleaning = false
			cache.m.Unlock()
			return
		}
		cache.m.Unlock()
	}
}

// validate checks the validity of all cache entries and deletes those that are
// expired.
//
// validate does not acquire a lock. It is the caller's responsibility to
// maintain a read/write lock on the cache during the call.
func (cache *Cache) validate(now time.Time) (next time.Time, more bool) {
	for len(cache.log) > 0 {
		expiration := cache.log[0].Timestamp.Add(cache.d)
		if expiration.After(now) {
			return expiration, true
		}

		k := cache.log[0].Key
		entry, found := cache.data[k]
		if found {
			// The expiration in the cache could be more recent than the log entry
			// we're processing, so it's important that we check it again.
			expiration = entry.Timestamp.Add(cache.d)
			if expiration.Before(now) {
				release(entry.Value)
				delete(cache.data, k)
			}
		}

		cache.log = cache.log[1:]
	}
	return time.Time{}, false
}

func release(v Value) {
	if r, ok := v.(Releaser); ok {
		go r.Release()
	} else if c, ok := v.(Closer); ok {
		go c.Close()
	} else if c, ok := v.(io.Closer); ok {
		go c.Close()
	}
}
