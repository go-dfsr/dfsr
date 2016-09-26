package callstat

import "time"

// Call represents information about a function call.
type Call struct {
	// Start is the time at which the call started.
	Start time.Time
	// End is the time at which the call ended or timed out.
	End time.Time
	// Err is the error returned from unsuccessful calls.
	Err error
}

// Duration is the total time it took to make the call.
func (c *Call) Duration() time.Duration {
	return c.End.Sub(c.Start)
}
