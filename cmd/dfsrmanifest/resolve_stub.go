// +build !windows

package main

import (
	"errors"

	"gopkg.in/dfsr.v0/core"
)

func resolve(domain string) (dom string, data core.Domain, err error) {
	dom = domain
	err = errors.New("domain resolution not supported on non-windows platforms")
	return
}
