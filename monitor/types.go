package monitor

import (
	"time"

	"gopkg.in/dfsr.v0/core"
)

// Source represents a domain-wide configuration source.
type Source interface {
	Value() (*core.Domain, time.Time, error)
}
