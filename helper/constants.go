package helper

import "errors"

var (
	// ErrClosed is returned from calls to a service or interface in the event
	// that the Close() function has already been called.
	ErrClosed = errors.New("Interface is closing or already closed.")
	// ErrZeroWorkers is returned when zero workers are specified in a call to
	// NewLimiter.
	ErrZeroWorkers = errors.New("Zero workers were specified for the limiter.")
)
