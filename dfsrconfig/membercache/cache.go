package membercache

import (
	"sync"

	"gopkg.in/dfsr.v0/core"
)

// Cache represents a threadsafe DFSR member configuration cache.
type Cache struct {
	m     sync.RWMutex
	cache map[string]core.MemberInfo
}

// New returns a new threadsafe DFSR member configuration cache.
func New() *Cache {
	return &Cache{
		cache: make(map[string]core.MemberInfo),
	}
}

// Set saves the given DFSR member configuration data in the cache.
func (mc *Cache) Set(member core.MemberInfo) {
	mc.m.Lock()
	defer mc.m.Unlock()
	mc.cache[member.DN] = member
}

// Retrieve returns the cached DFSR member configuration data for the given
// distinguished name. If the data is not present in the cache then ok will be
// false.
func (mc *Cache) Retrieve(dn string) (member core.MemberInfo, ok bool) {
	mc.m.RLock()
	defer mc.m.RUnlock()
	member, ok = mc.cache[dn]
	return
}
