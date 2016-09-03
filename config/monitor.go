package config

import (
	"sync"
	"time"

	"gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/core"
)

// Monitor polls Active Directory for updated domain-wide DFSR configuration.
type Monitor struct {
	mutex     sync.RWMutex
	domain    string
	interval  time.Duration
	cfg       *core.Domain
	updated   time.Time
	listeners []chan *core.Domain
	ready     sync.WaitGroup
	updating  bool
	pulse     chan struct{} // Signals update. nil indicates closed.
	stop      chan struct{} // Signals stop. nil indicates stopped.
}

// NewMonitor returns a new DFSR configuration monitor that polls Active
// Directory for updated DFSR configuration.
func NewMonitor(domain string, interval time.Duration) *Monitor {
	m := &Monitor{
		domain:   domain,
		interval: interval,
		pulse:    make(chan struct{}),
	}
	m.ready.Add(1)
	return m
}

// Close will release resources consumed by the monitor. It should be called
// when finished with the monitor. Calling Close will prevent future calls to
// Start or Update from succeeding.
func (m *Monitor) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.pulse == nil {
		return // Already closed
	}
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	m.pulse = nil // Note: do not close pulse here because it could trigger an update in run()
}

// Start starts the configuration monitor. If the monitor is already running
// Start does nothing and returns nil. If it is unable to initialize an ADSI
// client Start will return an error. If the monitor is already closed
// ErrClosed will be returned.
func (m *Monitor) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.pulse == nil {
		return ErrClosed
	}
	if m.stop != nil {
		return nil // Already running
	}
	client, err := adsi.NewClient()
	if err != nil {
		return err
	}
	m.stop = make(chan struct{})
	go m.run(client, m.domain, m.interval, m.pulse, m.stop)
	return nil
}

// Stop stops the monitor and prevents further polling of Active Directory
// until Start is called again.
func (m *Monitor) Stop() {
	m.mutex.Lock()
	if m.stop != nil {
		for _, ch := range m.listeners {
			close(ch)
		}
		m.listeners = nil

		close(m.stop)
		m.stop = nil
	}
	m.mutex.Unlock()
}

// Update requests immediate retrieval of configuration data from Active
// Directory. It does not wait for the retrieval to complete.
//
// If the monitor has not been started Update will do nothing.
func (m *Monitor) Update() {
	m.mutex.Lock()
	if m.pulse != nil && m.stop != nil {
		m.pulse <- struct{}{}
		return
	}
	m.mutex.Unlock()
}

// Config returns the most recently retrieved domain configuration data, or nil
// if it has not yet acquired any data.
func (m *Monitor) Config() (cfg *core.Domain) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.cfg
}

// WaitReady blocks until the monitor has retrieved configuration data. If the
// monitor has already retrieved data the call will not block.
func (m *Monitor) WaitReady() {
	m.ready.Wait()
}

// Listen returns a channel on which configuration updates will be broadcast.
// The channel will be closed when the monitor is stopped and will not be
// reopened.
func (m *Monitor) Listen() <-chan *core.Domain {
	ch := make(chan *core.Domain)
	m.mutex.Lock()
	m.listeners = append(m.listeners, ch)
	m.mutex.Unlock()
	return ch
}

func (m *Monitor) run(client *adsi.Client, domain string, interval time.Duration, pulse <-chan struct{}, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-pulse:
		case <-ticker.C:
		}

		go m.update(client, domain)
	}
}

func (m *Monitor) update(client *adsi.Client, domain string) {
	if !m.startUpdate() {
		// update goroutine is already running, don't spawn doubles
		return
	}

	updated := time.Now()
	cfg, err := Domain(client, domain)

	if err != nil {
		m.stopUpdate()
		// FIXME: Do something with the error, like sending it to a logger or a channel
		return
	}

	m.stopUpdateAndApply(updated, &cfg)
}

func (m *Monitor) startUpdate() (acquired bool) {
	m.mutex.Lock()
	if !m.updating {
		m.updating = true
		acquired = true
	}
	m.mutex.Unlock()
	return
}

func (m *Monitor) stopUpdate() {
	m.mutex.Lock()
	m.updating = false
	m.mutex.Unlock()
}

func (m *Monitor) stopUpdateAndApply(updated time.Time, cfg *core.Domain) {
	m.mutex.Lock()

	hadConfig := (m.cfg != nil)

	m.updated = updated
	m.cfg = cfg
	m.updating = false

	if !hadConfig && cfg != nil {
		m.ready.Done()
	}

	if len(m.listeners) == 0 {
		m.mutex.Unlock()
		return
	}

	recipients := append([]chan *core.Domain(nil), m.listeners...) // copy listeners for out-of-mutex processing
	m.mutex.Unlock()

	for _, r := range recipients {
		r <- cfg
	}
}
