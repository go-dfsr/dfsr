// +build windows

package main

// Error constants
const (
	_ = iota // 0 == success
	ErrGeneric
	ErrConfigInitFailure
	ErrBacklogInitFailure
)

// Event constants
const (
	EventInitProgress = iota
	EventInitComplete
	EventInitFailure
)
