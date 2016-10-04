package callstat

import (
	"fmt"
	"time"
)

// Call represents information about a function call.
type Call struct {
	// Description is a textual description of the call.
	Description string
	// Start is the time at which the call started.
	Start time.Time
	// End is the time at which the call ended or timed out.
	End time.Time
	// Err is the error returned from unsuccessful calls.
	Err error
	// Inner is the set of internal subcalls that were made as part of the call.
	Inner []Call
}

// Duration is the total time it took to make the call.
func (c *Call) Duration() time.Duration {
	return c.End.Sub(c.Start)
}

// Begin sets the call description and records the current time as the start of
// the call.
func (c *Call) Begin(description string) {
	c.Description = description
	c.Start = time.Now()
}

// Complete sets the call error state and records the current time as the end of
// the call.
//
// If the call was not previously started, Complete assumes that the start and
// end times of the call are equal.
func (c *Call) Complete(err error) {
	c.End = time.Now()
	c.Err = err
	if c.Start.IsZero() {
		c.Start = c.End
	}
}

// Add adds the given call as inner call.
func (c *Call) Add(inner *Call) {
	c.Inner = append(c.Inner, *inner)
}

// String returns a string representation of the call data.
func (c Call) String() string {
	desc := c.Description
	if desc == "" {
		desc = "Call"
	}

	seconds := fmt.Sprintf("%.4fs", c.Duration().Seconds())

	if len(c.Inner) == 0 {
		return fmt.Sprintf("%s %s", desc, seconds)
	}

	var inner []byte
	for i := 0; i < len(c.Inner); i++ {
		if i > 0 {
			inner = append(inner, ", "...)
		}
		inner = append(inner, c.Inner[i].String()...)
	}

	return fmt.Sprintf("%s %s (%s)", desc, seconds, inner)
}
