package config

import (
	"context"
	"sync"
	"time"

	"gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/core"
	"gopkg.in/dfsr.v0/poller"
	"gopkg.in/dfsr.v0/valuesink"
)

// domainBroadcaster broadcasts domain configuration updates to a set of
// listeners.
type domainBroadcaster struct {
	mutex     sync.RWMutex
	listeners []chan<- DomainUpdate
	closed    bool
}

func (bc *domainBroadcaster) Listen() <-chan DomainUpdate {
	ch := make(chan DomainUpdate, updateChanSize)
	bc.mutex.Lock()
	if !bc.closed {
		bc.listeners = append(bc.listeners, ch)
	} else {
		close(ch)
	}
	bc.mutex.Unlock()
	return ch
}

func (bc *domainBroadcaster) Broadcast(domain *core.Domain, timestamp time.Time, err error) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if len(bc.listeners) == 0 {
		return
	}

	for _, listener := range bc.listeners {
		listener <- DomainUpdate{
			Domain:    domain,
			Timestamp: timestamp,
			Err:       err,
		}
	}
}

func (bc *domainBroadcaster) Close() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if bc.closed {
		return
	}
	bc.closed = true

	for _, ch := range bc.listeners {
		close(ch)
	}
	bc.listeners = nil
}

// domainSource acts as a polling source for poller.Poller. It retrieves
// domain configuration data, updates a sink and sends data via a broadcaster.
type domainSource struct {
	client *adsi.Client
	domain string
	sink   *valuesink.Sink
	bc     *domainBroadcaster
}

func (ds *domainSource) Poll(ctx context.Context) {
	// FIXME: Propagate the context
	timestamp := time.Now()
	cfg, err := Domain(ds.client, ds.domain)
	ds.sink.Update(&cfg, timestamp, err)
	ds.bc.Broadcast(&cfg, timestamp, err)
}

func (ds *domainSource) Close() {
	ds.client.Close()
}

// DomainUpdate represents an update to domain configuration data.
type DomainUpdate struct {
	Domain    *core.Domain
	Timestamp time.Time
	Err       error
}

// DomainMonitor polls Active Directory for updated domain-wide DFSR
// configuration.
type DomainMonitor struct {
	sink valuesink.Sink    // Holds last configuration successfully retrieved
	bc   domainBroadcaster // Broadcasts configuration updates

	mutex    sync.Mutex
	domain   string
	interval time.Duration
	instance *poller.Poller
	closed   bool
}

// NewDomainMonitor returns a new DFSR configuration monitor that polls Active
// Directory for updated DFSR configuration for a domain. If the provided domain
// is an empty string the monitor will attempt to use the the domain of the
// computer it is running on by querying the root domain naming context.
func NewDomainMonitor(domain string, interval time.Duration) *DomainMonitor {
	m := &DomainMonitor{
		domain:   domain,
		interval: interval,
	}
	return m
}

// Close will release resources consumed by the monitor. It should be called
// when finished with the monitor. Calling close will prevent future calls to
// start or update from succeeding. Close will not return until all
// monitor-related goroutines have exited.
func (m *DomainMonitor) Close() {
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

// Start starts the configuration monitor. If the monitor is already running
// start does nothing and returns nil. If it is unable to initialize an ADSI
// client start will return an error. If the monitor is already closed
// ErrClosed will be returned.
func (m *DomainMonitor) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.closed {
		return ErrClosed
	}
	if m.instance != nil {
		return nil // Already running
	}

	client, err := adsi.NewClient()
	if err != nil {
		return err
	}
	if m.domain == "" {
		m.domain, err = dnc(client)
		if err != nil {
			return err
		}
		if m.domain == "" {
			return ErrDomainLookupFailed
		}
	}

	m.instance = poller.New(&domainSource{
		client: client,
		domain: m.domain,
		sink:   &m.sink,
		bc:     &m.bc,
	}, m.interval)

	return nil
}

// Stop stops the monitor and prevents further polling of Active Directory
// until Start is called again.
func (m *DomainMonitor) Stop() {
	m.mutex.Lock()
	if m.instance != nil {
		m.instance.Close() // TODO: Decide whether blocking here is acceptable
		m.instance = nil
	}
	m.mutex.Unlock()
}

// Value returns the most recently retrieved domain configuration data, or nil
// if it has not yet acquired any data.
func (m *DomainMonitor) Value() (cfg *core.Domain, timestamp time.Time, err error) {
	v, timestamp, err := m.sink.Value()
	cfg = v.(*core.Domain)
	return
}

// Listen returns a channel on which configuration updates will be broadcast.
// The channel will be closed when the monitor is closed. If the monitor has
// already been closed then the returned channel will be closed already.
func (m *DomainMonitor) Listen() <-chan DomainUpdate {
	return m.bc.Listen()
}

// WaitReady blocks until the monitor has retrieved configuration data. If the
// monitor has already retrieved data the call will not block.
func (m *DomainMonitor) WaitReady() (err error) {
	return m.sink.WaitReady()
}

// Update requests immediate retrieval of configuration data from Active
// Directory. It does not wait for the retrieval to complete.
//
// If the monitor has not been started Update will do nothing.
func (m *DomainMonitor) Update() {
	m.mutex.Lock()
	if !m.closed && m.instance != nil {
		m.instance.Poll()
	}
	m.mutex.Unlock()
}
