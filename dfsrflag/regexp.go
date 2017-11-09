package dfsrflag

import (
	"fmt"
	"regexp"
	"strings"
)

// RegexpSlice is a flag value that parses one or more regular expressions.
//
// All regular expressions parsed by this type will be case-insensitive.
type RegexpSlice []*regexp.Regexp

const regexci = "(?i)"

// String returns a string representation of the regular expression slice.
func (s *RegexpSlice) String() string {
	return fmt.Sprint(*s)
}

// Set compiles value and adds it to s. The regular expression will be
// case-insensitive.
//
// If the regular expression cannot be compiled an error will be returned.
func (s *RegexpSlice) Set(value string) error {
	if !strings.HasPrefix(value, regexci) {
		value = regexci + value
	}
	re, err := regexp.Compile(value)
	if err != nil {
		return err
	}
	*s = append(*s, re)
	return nil
}
