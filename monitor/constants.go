package monitor

import "errors"

const updateChanSize = 16

var (
	// ErrClosed is returned from calls to a service or interface in the event
	// that the Close() function has already been called.
	ErrClosed = errors.New("Monitor is closing or already closed.")
)
