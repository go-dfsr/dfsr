// +build !windows

package main

import (
	"errors"

	"gopkg.in/dfsr.v0/dfsr"
)

func resolve(domain string) (dom string, data dfsr.Domain, err error) {
	dom = domain
	err = errors.New("domain resolution not supported on non-windows platforms")
	return
}
