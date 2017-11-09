package mfilter

import (
	"time"

	"gopkg.in/dfsr.v0/manifest"
)

// After creates a filter that returns true when a resource's timestamp
// is after t.
func After(t time.Time) manifest.Filter {
	return func(r *manifest.Resource) bool {
		return r.Time.After(t)
	}
}

// Before creates a filter that returns true when a resource's timestamp
// is before t.
func Before(t time.Time) manifest.Filter {
	return func(r *manifest.Resource) bool {
		return r.Time.Before(t)
	}
}
