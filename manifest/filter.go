package manifest

// Filter returns true when a filter matches the given resource.
type Filter func(r *Resource) bool

// Match returns the result of f(r) if f is non-nil. When f is nil it returns
// true.
func (f Filter) Match(r *Resource) bool {
	if f == nil {
		return true
	}
	return f(r)
}
