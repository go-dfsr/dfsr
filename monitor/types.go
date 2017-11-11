package monitor

import (
	"time"

	"gopkg.in/dfsr.v0/dfsr"
)

// Source represents a domain-wide configuration source.
type Source interface {
	Value() (*dfsr.Domain, time.Time, error)
}
