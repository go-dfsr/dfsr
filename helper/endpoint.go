package helper

import (
	"context"
	"sync"
	"time"

	"github.com/gentlemanautomaton/calltracker"
	ole "github.com/go-ole/go-ole"
	"github.com/google/uuid"
	"gopkg.in/dfsr.v0/callstat"
	"gopkg.in/dfsr.v0/versionvector"
)

// DefaultEndpointConfig provides a default set of endpoint configuration
// values.
var DefaultEndpointConfig = EndpointConfig{
	Caching:       true,
	CacheDuration: time.Second * 30,
	Limiting:      true,
	Limit:         1,
	OnlineReconnectionInterval:  time.Minute * 30,
	OfflineReconnectionInterval: time.Minute * 2,
	AcceptableCallDuration:      time.Second * 30,
}

// EndpointConfig desribes a set of endpoint configuration parameters.
//
// Caching instructs the client to cache retrieved version vectors for a
// specified duration.
//
// Limiting instructs the client to limit the maximum number of simultaneous
// workers that can talk to an endpoint.
type EndpointConfig struct {
	Caching                     bool
	CacheDuration               time.Duration
	Limiting                    bool
	Limit                       uint          // Maximum number of simultaneous calls
	OnlineReconnectionInterval  time.Duration // Time between connection attempts when endpoint is online
	OfflineReconnectionInterval time.Duration // Time between connection attempts when endpoint is offline
	AcceptableCallDuration      time.Duration // Maximum amount of time a remote procedure call is allowed before it is considered unresponsive

	// TODO: Use ICMP pings to assess network failure
	//PingInterval  time.Duration
	//PingTolerance time.Duration // Maximum time to wait for ping reponses
	//PingCount     int           // Number of pings to send during each assessment
}

// EndpointState describes the current condition of an endpoint.
type EndpointState struct {
	Err       error
	Changed   time.Time         // Last time the state changed
	Updated   time.Time         // Last time the state was updated
	IdleSince time.Time         // Last time an action was performed on the endpoint
	Calls     calltracker.Value // Representation of outstanding calls
}

// Online returns true if the state indicates that the endpoint is online.
func (s *EndpointState) Online() bool {
	return s.Err == nil
}

// Unresponsive returns true if the state indicates that the endpoint is online
// but not responding promptly to requests.
//
// The provided theshold is the maximum amount of time that may elapse before
// a remote procedure call is considered unresponsive.
func (s *EndpointState) Unresponsive(threshold time.Duration) bool {
	if s.Calls.Len() == 0 {
		return false
	}

	return s.Calls.MaxElapsed() > threshold
}

// Closed returns true if the state indicates that the endpoint has been closed.
func (s *EndpointState) Closed() bool {
	return s.Err == ErrClosed
}

const endpointChanSize = 32

//var _ = (Reporter)((*Endpoint)(nil)) // Compile-time interface compliance check

// Endpoint manages a connection to a remote or local server that implements the
// DFSR Helper protocol. It monitors the health of the connection by checking
// the errors returned by all queries for RPC connection failures.
//
// When a connection is determined to be offline the endpoint manager will
// proactively attempt to reestablish it on a configurable interval. While
// offline all queries will fail immediately.
//
// The underlying connection is reset periodically even when the connection is
// healthy in order to release resources in the RPC layer of remote systems that
// are encumbered by memory leaks. The online reconnection interval is also
// configurable.
//
// The zero value of an endpoint is not suitable for use. Endpoints should be
// created with a call to NewEndpoint().
//
// When finished with an endpoint, it is necessary to call Close() to release
// any resources consumed by the endpoint and to stop monitoring the health of
// its connection.
type Endpoint struct {
	fqdn         string
	ready        sync.WaitGroup      // Marks completion of first connection attempt
	closed       sync.WaitGroup      // Marks the exit of run()
	configChange chan EndpointConfig // Receives configuration updates. Consumed by run(). Closure initiates shutdown.
	stateChange  chan EndpointState  // Receives state changes. Consumed by run(). Closure initiates shutdown.
	tracker      calltracker.Tracker // Tracks the number and condition of outstanding remote procedure calls.

	mutex    sync.RWMutex
	config   EndpointConfig
	state    EndpointState
	sequence uint64 // Last health update sequence number received
	r        Reporter
}

// NewEndpoint creates a new endpoint and returns it without blocking. The
// returned endpoint will be initialized asynchronously in its own goroutine.
func NewEndpoint(fqdn string, config EndpointConfig) *Endpoint {
	now := time.Now()
	e := &Endpoint{
		fqdn:         fqdn,
		configChange: make(chan EndpointConfig, endpointChanSize),
		stateChange:  make(chan EndpointState, endpointChanSize),
		config:       config,
		state: EndpointState{
			Err:     ErrDisconnected,
			Changed: now,
			Updated: now,
		},
	}
	e.ready.Add(1)
	e.closed.Add(1)
	state := e.state
	e.tracker.Subscribe(e.updateHealth)
	go e.run(config, state)
	return e
}

// Config returns the current configuration of the endpoint.
func (e *Endpoint) Config() (config EndpointConfig) {
	e.mutex.RLock()
	config = e.config
	e.mutex.RUnlock()
	return
}

// UpdateConfig updates the endpoint configuration.
func (e *Endpoint) UpdateConfig(config EndpointConfig) {
	e.mutex.Lock()
	e.config = config
	if !e.state.Closed() {
		e.configChange <- config
	}
	e.mutex.Unlock()
}

// State returns the current state of the endpoint.
func (e *Endpoint) State() (state EndpointState) {
	e.mutex.RLock()
	state = e.state
	e.mutex.RUnlock()
	return
}

// Close releases any resources consumed by the endpoint.
func (e *Endpoint) Close() {
	e.mutex.Lock()
	if e.state.Err == ErrClosed {
		e.mutex.Unlock()
		return
	}
	e.state.Err = ErrClosed
	// Closing either of these channels causes run() to exit
	close(e.configChange)
	close(e.stateChange)
	e.mutex.Unlock()

	// Wait for run() to wrap up so that it can't recreate the connection after
	// we close it
	e.closed.Wait()

	e.mutex.Lock()
	if e.r != nil {
		e.r.Close()
		e.r = nil
	}
	e.mutex.Unlock()
}

// Vector returns the reference version vectors for the requested replication
// group.
func (e *Endpoint) Vector(ctx context.Context, group uuid.UUID) (vector *versionvector.Vector, call callstat.Call, err error) {
	call.Begin("Endpoint.Vector")
	defer call.Complete(err)

	e.ready.Wait()
	e.mutex.RLock()
	r, state, threshold, err := e.r, e.state, e.config.AcceptableCallDuration, e.state.Err
	e.mutex.RUnlock()

	if err != nil {
		return
	}

	if state.Unresponsive(threshold) {
		err = ErrUnresponsive
		return
	}

	var subcall callstat.Call
	vector, subcall, err = r.Vector(ctx, group, &e.tracker)
	call.Add(&subcall)

	e.updateStateAfterCall(r, err, time.Now())
	return
}

// Backlog returns the current backlog when compared against the given
// reference version vector.
func (e *Endpoint) Backlog(ctx context.Context, vector *versionvector.Vector) (backlog []int, call callstat.Call, err error) {
	call.Begin("Endpoint.Backlog")
	defer call.Complete(err)

	e.ready.Wait()
	e.mutex.RLock()
	r, state, threshold, err := e.r, e.state, e.config.AcceptableCallDuration, e.state.Err
	e.mutex.RUnlock()

	if err != nil {
		return
	}

	if state.Unresponsive(threshold) {
		err = ErrUnresponsive
		return
	}

	var subcall callstat.Call
	backlog, subcall, err = r.Backlog(ctx, vector, &e.tracker)
	call.Add(&subcall)

	e.updateStateAfterCall(r, err, time.Now())
	return
}

// Report generates a report when compared against the reference version vector.
func (e *Endpoint) Report(ctx context.Context, group uuid.UUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, call callstat.Call, err error) {
	call.Begin("Endpoint.Report")
	defer call.Complete(err)

	e.ready.Wait()
	e.mutex.RLock()
	r, err := e.r, e.state.Err
	e.mutex.RUnlock()

	if err != nil {
		return
	}

	var subcall callstat.Call
	data, report, subcall, err = r.Report(ctx, group, vector, backlog, files)
	call.Add(&subcall)

	e.updateStateAfterCall(r, err, time.Now())
	return
}

func (e *Endpoint) run(config EndpointConfig, state EndpointState) {
	// run relies on a timer to signal connection and reconnection.
	//
	// The timer will be continuously reset so that acts like a ticker, but we
	// don't use a ticker here for two reasons:
	//
	// 1. Connection attempts can take a long time to timeout and we want the
	//    connection interval to exclude that time.
	// 2. The connection interval changes depending on whether the endpoint is
	//    online or offline.

	defer e.closed.Done()

	var (
		//healthTimer   = time.NewTimer(0) // Triggers health evaluation to see if calls are responding quickly
		connTimer     = time.NewTimer(0) // Triggers new connections
		connTimestamp time.Time          // Last time the connection was reset
		initialized   bool
	)
	defer connTimer.Stop()

	for {
		select {
		case newConfig, ok := <-e.configChange:
			if !ok {
				return // endpoint is closing
			}

			var (
				cacheChange     = config.Caching != newConfig.Caching || config.CacheDuration != newConfig.CacheDuration
				limitChange     = config.Limiting != newConfig.Limiting || config.Limit != newConfig.Limit
				connTimerChange = config.OfflineReconnectionInterval != newConfig.OfflineReconnectionInterval || config.OnlineReconnectionInterval != newConfig.OnlineReconnectionInterval
			)

			config = newConfig

			switch {
			case cacheChange || limitChange:
				resetActiveTimer(connTimer, 0) // Reconnect to apply new configuration
			case connTimerChange:
				resetConnectionTimer(connTimer, state.Online(), &config, connTimestamp)
			}
		case newState, ok := <-e.stateChange:
			if !ok {
				return // endpoint is closing
			}

			onlineChange := state.Online() != newState.Online()

			state = newState

			if onlineChange && !state.Online() {
				resetActiveTimer(connTimer, 0) // Try to reconnect immediately
			}
		case <-connTimer.C:
			var (
				r         Reporter
				err       error
				makeReady bool
			)
			r, connTimestamp, err = createEndpointConnection(e.fqdn, config)
			if !initialized {
				initialized = true
				makeReady = true
			}
			go e.updateConnection(r, err, connTimestamp, makeReady)

			if err == nil {
				connTimer.Reset(config.OnlineReconnectionInterval)
			} else {
				connTimer.Reset(config.OfflineReconnectionInterval)
			}
			/*
				case <-healthTimer.C:
					go e.updateHealth()
					healthTimer.Reset()
			*/
		}
	}
}

// updateConnectionState will update the endpoint's error state.
//
// The caller must hold a write lock on the endpoint during the function call.
func (e *Endpoint) updateConnectionState(err error, when time.Time, onlyIfNewer bool) {
	if e.state.Closed() {
		return
	}

	if onlyIfNewer && e.state.Updated.After(when) {
		return
	}

	if e.state.Err != err {
		e.state.Changed = when
	}
	e.state.Err = err
	e.state.Updated = when

	e.stateChange <- e.state
}

func (e *Endpoint) updateConnection(r Reporter, err error, when time.Time, makeReady bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var retired Reporter

	if e.state.Closed() {
		retired = r
	} else {
		retired, e.r, e.state.Err = e.r, r, err
		e.updateConnectionState(err, when, false)
	}

	if retired != nil {
		go retired.Close()
	}

	if makeReady {
		e.ready.Done()
	}
}

// updateStateAfterCall will evaluate the provided err to determine whether
// it indicates a change in the state of the endpoint. If so, it will record the
// state change. It will also update the endpoint's idle time.
func (e *Endpoint) updateStateAfterCall(r Reporter, err error, when time.Time) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.state.IdleSince.Before(when) {
		e.state.IdleSince = when
	}

	if IsUnavailableErr(err) {
		// The error only affects the current state if it's for the current
		// connection. The connection could have been reset while this call was
		// being made.
		if e.r == r {
			e.updateConnectionState(err, when, true)
		}
	}
}

// updateHealth will update the endpoint's health state as provided by
// its call tracker.
func (e *Endpoint) updateHealth(update calltracker.Update) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.state.Closed() {
		return
	}

	if e.sequence > update.Sequence {
		// Already have newer values
		return
	}

	e.state.Updated = time.Now()
	e.state.Calls = update.Value
	e.sequence = update.Sequence

	e.stateChange <- e.state
}

func createEndpointConnection(fqdn string, config EndpointConfig) (r Reporter, timestamp time.Time, err error) {
	timestamp = time.Now()

	r, err = NewReporter(fqdn)
	if err != nil {
		return
	}

	if config.Limiting {
		rep := r
		r, err = NewLimiter(r, config.Limit)
		if err != nil {
			rep.Close()
			return
		}
	}

	if config.Caching {
		r = NewCacher(r, config.CacheDuration)
	}

	return
}

func resetConnectionTimer(t *time.Timer, online bool, config *EndpointConfig, connTimestamp time.Time) {
	var interval time.Duration
	if online {
		interval = config.OnlineReconnectionInterval
	} else {
		interval = config.OfflineReconnectionInterval
	}
	d := connTimestamp.Add(interval).Sub(time.Now())
	resetActiveTimer(t, d)
}

func resetActiveTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		<-t.C
	}
	t.Reset(d)
}
