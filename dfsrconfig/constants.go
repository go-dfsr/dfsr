package dfsrconfig

import (
	"errors"
	"time"
)

const updateChanSize = 16
const groupQueryDelay = 25 * time.Millisecond // Group query delay to avoid rate-limiting by LDAP servers

var (
	// ErrClosed is returned from calls to a service or interface in the event
	// that the Close() function has already been called.
	ErrClosed = errors.New("interface is closing or already closed")

	// ErrDomainLookupFailed is returned when the appropriate domain naming
	// context cannot be determined.
	ErrDomainLookupFailed = errors.New("unable to determine DFSR configuration domain")
)
