package valuesink

import (
	"sync"
	"time"
)

// Sink represents a threadsafe value container that can receive updates.
// It is intended to be used in conjuction with an external trigger or
// polling function that updates the contained value.
//
// Sink always stores the last non-nil value provided via update for which there
// was no associated error. This value can be retrieved by calling the value
// function.
//
// It is safe to intialize a sink with its zero value or to embed a sink in
// other types.
type Sink struct {
	mutex     sync.RWMutex
	value     interface{}
	timestamp time.Time
	err       error // Passes errors to WaitReady()
	ready     *sync.WaitGroup
	closed    bool
}

// Close releases any resources consumed by the sink.
func (s *Sink) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed {
		return
	}

	s.closed = true

	if s.ready != nil {
		s.ready.Done()
		s.ready = nil
	}
}

// Value returns the current value if the sink is not empty, otherwise it
// returns nil.
//
// Calls to this function are guaranteed to succeed if the sink is ready, as
// indicated by Sink.Ready().
func (s *Sink) Value() (value interface{}, timestamp time.Time, err error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.value, s.timestamp, s.err
}

// Ready returns true if the sink contains a value, otherwise it returns
// false.
func (s *Sink) Ready() (ready bool) {
	s.mutex.RLock()
	ready = s.value != nil
	s.mutex.RUnlock()
	return
}

// WaitReady blocks until the next call to Update if the sink isn't ready,
// otherwise WaitReady does nothing and does not block.
//
// WaitReady returns nil when the sink is ready with a value. If a call to
// Update provides an error, WaitReady unblocks and returns that error.
//
// If the sink does not contain a value and has been closed then ErrClosed will
// be returned.
func (s *Sink) WaitReady() (err error) {
	s.mutex.Lock()

	if s.value != nil {
		s.mutex.Unlock()
		return nil
	}

	if s.closed {
		s.mutex.Unlock()
		return ErrClosed
	}

	if s.ready == nil {
		s.ready = new(sync.WaitGroup)
		s.ready.Add(1)
	}
	rdy := s.ready

	s.mutex.Unlock()

	rdy.Wait()

	s.mutex.RLock()
	if s.value == nil && s.err == nil && s.closed {
		err = ErrClosed
	} else {
		err = s.err
	}
	s.mutex.RUnlock()

	return
}

// Update will update the value contained in the sink and unblock any
// oustanding calls to WaitReady.
func (s *Sink) Update(value interface{}, timestamp time.Time, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return
	}

	initial := (s.value == nil)

	if err == nil {
		s.value = value
	}

	if err == nil || initial {
		s.timestamp = timestamp
		s.err = err
	}

	if s.ready != nil {
		// This notifies callers waiting for the initial value to become available.
		// This notification is sent even if err != nil.
		s.ready.Done()
		s.ready = nil
	}
}
