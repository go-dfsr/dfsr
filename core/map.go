package core

// MemberInfoMap maps member identifiers to member information.
type MemberInfoMap map[string]MemberInfo

// Resolve returns information about the member with the given key, which is
// typically a GUID in lower case.
func (m MemberInfoMap) Resolve(key string) (info MemberInfo, ok bool) {
	info, ok = m[key]
	return
}
