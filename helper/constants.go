package helper

import (
	"errors"
	"time"
)

const (
	// DefaultRecoveryInterval specifies the default recovery interval for
	// client instances.
	DefaultRecoveryInterval = time.Second * 30
)

var (
	// ErrDisconnected is returned when a server is offline.
	ErrDisconnected = errors.New("the server is disconnected or offline")
	// ErrClosed is returned from calls to a service or interface in the event
	// that the Close() function has already been called.
	ErrClosed = errors.New("interface is closing or already closed")
	// ErrUnresponsive is returned from calls to a service or interface when
	// the underlying remote procedure call stalls for an unreasonable length of
	// time.
	ErrUnresponsive = errors.New("the server is unresponsive")
	// ErrZeroWorkers is returned when zero workers are specified in a call to
	// NewLimiter.
	ErrZeroWorkers = errors.New("no workers were specified for the limiter")
)
