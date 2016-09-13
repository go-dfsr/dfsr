package monitor

import (
	"sync"
	"time"

	"gopkg.in/dfsr.v0/core"
	"gopkg.in/dfsr.v0/helper"
	"gopkg.in/dfsr.v0/poller"
	"gopkg.in/dfsr.v0/valuesink"
)

// Monitor represents a DFSR backlog monitor for a domain.
type Monitor struct {
	sink valuesink.Sink     // Will hold last known global current state. Not yet used.
	bc   backlogBroadcaster // Broadcasts configuration updates

	mutex    sync.Mutex
	source   Source
	interval time.Duration
	cache    time.Duration
	limit    uint
	instance *poller.Poller
	closed   bool
}

// New creates a new Monitor with the given source and polling interval.
//
// If cache is nonzero then the monitor will cache version vectors for
// the given duration.
//
// If limit is nonzero then the monitor will limit the number of active
// queries to an individual DFSR member to the given value.
//
// The returned monitor will not function until start is called.
func New(source Source, interval time.Duration, cache time.Duration, limit uint) *Monitor {
	return &Monitor{
		source:   source,
		interval: interval,
		cache:    cache,
		limit:    limit,
	}
}

// Close will release resources consumed by the monitor. It should be called
// when finished with the monitor.
func (m *Monitor) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return
	}
	m.closed = true

	if m.instance != nil {
		m.instance.Close() // Blocks until the instance completely winds down
		m.instance = nil
	}

	m.sink.Close()
	m.bc.Close()
}

// Start starts the monitor. If the monitor is already running start does
// nothing and returns nil. If it is unable to initialize a DFSR client
// start will return an error. If the monitor is already closed
// ErrClosed will be returned.
func (m *Monitor) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.closed {
		return ErrClosed
	}
	if m.instance != nil {
		return nil // Already running
	}

	client, err := helper.NewClient()
	if err != nil {
		return err
	}
	if m.cache > time.Duration(0) {
		client.Cache(m.cache)
	}
	if m.limit > 0 {
		client.Limit(m.limit)
	}

	m.instance = poller.New(&worker{
		client: client,
		source: m.source,
		sink:   &m.sink,
		bc:     &m.bc,
	}, m.interval)

	return nil
}

// Stop stops the monitor and prevents further polling of DFSR backlogs until
// start is called again.
func (m *Monitor) Stop() {
	m.mutex.Lock()
	if m.instance != nil {
		m.instance.Close() // TODO: Decide whether blocking here is acceptable
		m.instance = nil
	}
	m.mutex.Unlock()
}

// Update requests immediate retrieval of DFSR backlogs. It does not wait for
// the retrieval to complete.
//
// If the monitor has not been started Exec will do nothing. If an update is
// already running a second update will not be started.
func (m *Monitor) Update() {
	m.mutex.Lock()
	if !m.closed && m.instance != nil {
		m.instance.Poll()
	}
	m.mutex.Unlock()
}

// Listen returns a channel on which DFSR backlog values will be broadcast.
// The channel will be closed when the monitor is closed.
func (m *Monitor) Listen() <-chan *core.Backlog {
	return m.bc.Listen()
}
