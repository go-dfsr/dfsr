package config

import (
	"sync"

	"gopkg.in/dfsr.v0"
)

type memberCache struct {
	m     sync.RWMutex
	cache map[string]dfsr.MemberInfo
}

func newMemberCache() *memberCache {
	return &memberCache{
		cache: make(map[string]dfsr.MemberInfo),
	}
}

func (mc *memberCache) Add(member dfsr.MemberInfo) {
	mc.m.Lock()
	defer mc.m.Unlock()
	mc.cache[member.DN] = member
}

func (mc *memberCache) Retrieve(dn string) (member dfsr.MemberInfo, ok bool) {
	mc.m.RLock()
	defer mc.m.RUnlock()
	member, ok = mc.cache[dn]
	return
}
