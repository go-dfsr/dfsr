// +build windows

package main

// Windows Service Properties
const (
	DefaultServiceName = "dfsrmonitor"
	DefaultDisplayName = "DFSR Monitor"
	DefaultDescription = "Monitors DFSR backlog counts"
)

// Error constants
const (
	_ = iota // 0 == success
	ErrGeneric
	ErrConfigInitFailure
	ErrBacklogInitFailure
)

// Event constants
const (
	EventInitProgress = iota + 1
	EventInitComplete
	EventInitFailure
)
