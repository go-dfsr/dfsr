package mfilter

import (
	"gopkg.in/dfsr.v0/manifest"
)

// And creates a filter that returns true when all of the provided filters
// return true.
//
// If filters is empty, nil, or composed solely of nil filters, a nil filter
// will be returned.
func And(filters ...manifest.Filter) manifest.Filter {
	filters = selectNonNil(filters)
	if len(filters) == 0 {
		return nil
	}
	return func(r *manifest.Resource) bool {
		for _, filter := range filters {
			if !filter.Match(r) {
				return false
			}
		}
		return true
	}
}

// Or creates a filter that returns true when at least one of the provided
// filters return true.
//
// If filters is empty, nil, or composed solely of nil filters, a nil filter
// will be returned.
func Or(filters ...manifest.Filter) manifest.Filter {
	filters = selectNonNil(filters)
	if len(filters) == 0 {
		return nil
	}
	return func(r *manifest.Resource) bool {
		for _, filter := range filters {
			if filter.Match(r) {
				return true
			}
		}
		return false
	}
}

// Not creates a filter that returns true when the given filter does not.
func Not(filter manifest.Filter) manifest.Filter {
	return func(r *manifest.Resource) bool {
		return !filter.Match(r)
	}
}

// selectNonNil returns a copy of filters that only includes non-nil members.
func selectNonNil(filters []manifest.Filter) []manifest.Filter {
	selected := make([]manifest.Filter, 0, len(filters))
	for _, filter := range filters {
		if filter != nil {
			selected = append(selected, filter)
		}
	}
	if len(selected) == 0 {
		return nil
	}
	return selected
}
