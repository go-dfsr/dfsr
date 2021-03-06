package manifest

import "gopkg.in/dfsr.v0/dfsr"

// Resolver is capable of resolving DFSR member GUIDs.
type Resolver func(guid string) (info dfsr.MemberInfo, ok bool)

// Resolve returns the result of r(guid) if r is non-nil. If r is nil it
// returns false.
func (r Resolver) Resolve(guid string) (info dfsr.MemberInfo, ok bool) {
	if r == nil {
		return
	}
	return r(guid)
}
