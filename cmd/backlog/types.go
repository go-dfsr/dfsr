package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type regexSlice []*regexp.Regexp

const regexci = "(?i)"

func (s *regexSlice) String() string {
	return fmt.Sprint(*s)
}

func (s *regexSlice) Set(value string) error {
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

type uintOrInf struct {
	Present bool
	Inf     bool
	Value   uint
}

func (u *uintOrInf) String() string {
	if u.Inf {
		return "infinite"
	}
	return fmt.Sprint(u.Value)
}

func (u *uintOrInf) Set(value string) (err error) {
	u.Present = true

	switch strings.ToLower(value) {
	case "infinite", "inf", "i":
		u.Inf = true
	default:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("\"%s\" is neither an acceptable number nor \"infinite\"", value)
		}
		u.Value = uint(v)
	}

	return
}
