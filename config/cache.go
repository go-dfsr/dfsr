package config

import (
	"sync"

	"gopkg.in/dfsr.v0/core"
)

type memberCache struct {
	m     sync.RWMutex
	cache map[string]core.MemberInfo
}

func newMemberCache() *memberCache {
	return &memberCache{
		cache: make(map[string]core.MemberInfo),
	}
}

func (mc *memberCache) Add(member core.MemberInfo) {
	mc.m.Lock()
	defer mc.m.Unlock()
	mc.cache[member.DN] = member
}

func (mc *memberCache) Retrieve(dn string) (member core.MemberInfo, ok bool) {
	mc.m.RLock()
	defer mc.m.RUnlock()
	member, ok = mc.cache[dn]
	return
}
