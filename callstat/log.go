package callstat

import "time"

// Log represents information about one or more function calls.
//
// Log is designed to avoid unneccessary memory management. It is intended to be
// passed by value and not by reference.
type Log struct {
	single  [1]Call
	entries []Call
}

// Single returns a call log with a single call in it.
func Single(start, end time.Time, err error) (log Log) {
	return Log{
		single: [1]Call{{
			Start: start,
			End:   end,
			Err:   err,
		}},
	}
}

// Entries returns the set of entries contained in the call log.
func (log *Log) Entries() []Call {
	if log.single[0].End.IsZero() {
		return nil
	}
	if log.entries == nil {
		return log.single[:]
	}
	return log.entries
}

// Set replaces the call log with the given call.
func (log *Log) Set(start, end time.Time, err error) {
	log.single[0] = Call{Start: start, End: end, Err: err}
	log.entries = nil
}

// Append appends a call to the call log.
func (log *Log) Append(start, end time.Time, err error) {
	call := Call{Start: start, End: end, Err: err}
	if log.single[0].End.IsZero() {
		log.single[0] = call
	} else if log.entries == nil {
		log.entries = append(log.single[:], call)
	} else {
		log.entries = append(log.entries, call)
	}
}

// Start prepares a new log entry and records the current time as the start
// time of that entry.
func (log *Log) Start() {
	start := time.Now()
	if log.single[0].End.IsZero() {
		log.single[0].Start = start
	} else if log.entries == nil {
		log.entries = append(log.single[:], Call{Start: start})
	} else {
		last := &log.entries[len(log.entries)-1]
		if last.End.IsZero() {
			last.Start = start
		} else {
			log.entries = append(log.entries, Call{Start: start})
		}
	}
}

// Complete completes a log entry that was created with start and records the
// current time as the end time. The provided error is recorded in the entry.
//
// If an entry was not previously started, Complete assumes that the start and
// end times of the call are equal.
func (log *Log) Complete(err error) {
	end := time.Now()
	if log.single[0].End.IsZero() {
		if log.single[0].Start.IsZero() {
			log.single[0].Start = end
		}
		log.single[0].End = end
		log.single[0].Err = err
	} else if log.entries == nil {
		log.entries = append(log.single[:], Call{Start: end, End: end, Err: err})
	} else {
		last := &log.entries[len(log.entries)-1]
		if last.End.IsZero() {
			if last.Start.IsZero() {
				last.Start = end
			}
			last.End = end
			last.Err = err
		} else {
			log.entries = append(log.entries, Call{Start: end, End: end, Err: err})
		}
	}
}
