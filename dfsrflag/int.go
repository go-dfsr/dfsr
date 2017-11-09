package dfsrflag

import (
	"fmt"
	"strconv"
	"strings"
)

// UintOrInf stores an unsigned integer value that can be used by the flag
// package. It is capable of indicating the special value infinity and whether
// a value was specified or not.
type UintOrInf struct {
	Present bool
	Inf     bool
	Value   uint
}

// String returns a string representation of the unsigned integer.
func (u *UintOrInf) String() string {
	if u.Inf {
		return "∞"
	}
	return fmt.Sprint(u.Value)
}

// Set parses the given value and assigns it to u.
//
// If value cannot be parsed an error will be returned.
func (u *UintOrInf) Set(value string) (err error) {
	u.Present = true

	switch strings.ToLower(value) {
	case "infinite", "infinity", "inf", "i", "∞":
		u.Inf = true
	default:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("\"%s\" is neither an unsigned integer nor a form of \"infinity\"", value)
		}
		u.Value = uint(v)
	}

	return
}
