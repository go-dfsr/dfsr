package mfilter

import (
	"regexp"

	"gopkg.in/dfsr.v0/manifest"
)

// PathRegexp creates a new filter that returns true when a resource's path
// matches the given regular expression.
func PathRegexp(re *regexp.Regexp) manifest.Filter {
	return func(r *manifest.Resource) bool {
		return re.MatchString(r.Path)
	}
}

// TypeRegexp creates a new filter that returns true when a resource's type
// matches the given regular expression.
func TypeRegexp(re *regexp.Regexp) manifest.Filter {
	return func(r *manifest.Resource) bool {
		return re.MatchString(r.Type)
	}
}
