package poller

import (
	"context"
	"sync"
	"time"
)

// Source is a polling source.
type Source interface {
	Poll(ctx context.Context)
	Close() // TODO: Consider removing this and doing a runtime type check for io.Closer
}

// Poller executes a polling function on an interval.
type Poller struct {
	interval time.Duration
	source   Source

	mutex  sync.Mutex
	cancel context.CancelFunc // Cancellation function. Nil when not running.
	pulse  chan struct{}      // Signals update. nil indicates closed.
	stop   chan struct{}      // Signals stop. nil indicates stopped.
	idle   *sync.Cond
	closed bool
}

// New returns a new poller for the given source. A ticker with the provided
// polling interval will be started immediately, but an invocation of the
// polling function will not run until its interval has elapsed. If immediate
// invocation is desired the Poll function should be called immediately after
// the poller has been created.
func New(source Source, interval time.Duration) *Poller {
	p := &Poller{
		source:   source,
		interval: interval,
		pulse:    make(chan struct{}),
		stop:     make(chan struct{}),
	}
	p.idle = sync.NewCond(&p.mutex)
	go p.run()
	return p
}

// Close causes the poller to stop polling and release any resources consumed
// by the poller. It will implicitly call the close function on the polling
// source.
func (p *Poller) Close() {
	p.mutex.Lock()
	// Don't defer p.mutex.Unlock() here because that would mess up sync.Cond.Wait
	if p.closed {
		p.mutex.Unlock()
		return
	}

	p.closed = true

	close(p.stop)
	close(p.pulse)

	if p.running() {
		p.cancel()
	}

	// If there's an update goroutine still running, wait until it's done before
	// closing the source.
	for p.running() {
		p.idle.Wait()
	}

	p.source.Close() // TODO: Consider doing a runtime interface type check here
	p.mutex.Unlock()
}

// Poll causes the poller to immediately poll the polling source. It does
// not wait for the polling action to complete.
func (p *Poller) Poll() {
	p.mutex.Lock()
	if !p.closed {
		p.pulse <- struct{}{}
	}
	p.mutex.Unlock()
}

func (p *Poller) run() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-p.pulse:
		case <-ticker.C:
		}

		go p.invoke()
	}
}

// running returns true if an update is running. It must be called while
// a lock on the poller's mutex is held.
func (p *Poller) running() bool {
	return p.cancel != nil
}

func (p *Poller) invoke() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !p.startInvocation(cancel) {
		// There is an update goroutine already running, so we're skipping this
		// tick so that we don't spawn doubles
		return
	}

	p.source.Poll(ctx)

	p.finishInvocation()
}

func (p *Poller) startInvocation(cancel context.CancelFunc) (acquired bool) {
	p.mutex.Lock()
	if !p.closed && !p.running() {
		p.cancel = cancel
		acquired = true
	}
	p.mutex.Unlock()
	return
}

func (p *Poller) finishInvocation() {
	p.mutex.Lock()
	p.cancel = nil
	p.mutex.Unlock()
	p.idle.Broadcast()
}
