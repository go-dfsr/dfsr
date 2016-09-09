package valuesink

import "errors"

var (
	// ErrClosed is returned from calls to a service or interface in the event
	// that the Close() function has already been called.
	ErrClosed = errors.New("Value sink is closing or already closed.")
)
