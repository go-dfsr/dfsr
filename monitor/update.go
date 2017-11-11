package monitor

import (
	"sync"
	"time"

	"gopkg.in/dfsr.v0/dfsr"
)

// Update represents an in-progress DFSR update performed by the monitor.
type Update struct {
	Domain *dfsr.Domain // Domain configuration at the time of the update
	//Connections []Connection

	size    int            // Number of backlog entries to be received in this update
	start   time.Time      // "locked" until started.Done()
	end     time.Time      // "locked" until ended.Done()
	started sync.WaitGroup // Have we received the start time?
	ended   sync.WaitGroup // Have we received the end time?

	mutex     sync.Mutex
	canceled  bool            // Has the update been canceled?
	values    []*dfsr.Backlog // "locked" until received.Done()
	received  sync.WaitGroup  // Have we received all of the backlog values?
	remaining int             // Number of backlog entries that have yet to be received for this update
	listeners []updateListener
}

// Listen returns a channel that receives the DFSR backlog values as they are
// retrieved for this update, in no particular order.
//
// The returned channel will be buffered with sufficient capacity to hold the
// maximum number of values that could be returned. Once all of the values for
// this update have been sent the channel will be closed. This makes the
// channel suitable for consumption with range:
//
//   for backlog := range update.Listen() {
//     log.Printf("Backlog retrieved: %d", backlog.Sum())
//   }
func (u *Update) Listen() <-chan *dfsr.Backlog {
	ch := make(chan *dfsr.Backlog, u.size)
	if u.size == 0 {
		close(ch)
		return ch
	}

	u.mutex.Lock()
	defer u.mutex.Unlock()

	// Send the values that we've already received
	for _, v := range u.values {
		ch <- v
	}

	if u.canceled {
		close(ch)
		return ch
	}

	i := len(u.listeners)
	u.listeners = append(u.listeners, updateListener{})
	u.listeners[i].init(ch, u.remaining)

	return ch
}

// Values will block until the update has finished, then return a slice of the
// retrieved backlog data.
func (u *Update) Values() (values []*dfsr.Backlog) {
	u.received.Wait()
	return u.values
}

// Size returns the number of values that will be returned in the update if it
// completes successfully. If the update is canceled the actual number of values
// returned may be less than this number.
func (u *Update) Size() int {
	return u.size
}

// Start will return the start time of the update.
func (u *Update) Start() time.Time {
	u.started.Wait()
	return u.start
}

// End will wait until the update has finished, then return the completion time
// of the update.
func (u *Update) End() time.Time {
	u.ended.Wait()
	return u.end
}

// Duration will wait until the update has finished, then return the total
// wall time of the update.
func (u *Update) Duration() time.Duration {
	u.ended.Wait()
	return u.end.Sub(u.start)
}

func newUpdate(domain *dfsr.Domain, size int) *Update {
	u := &Update{
		Domain:    domain,
		size:      size,
		remaining: size,
		listeners: make([]updateListener, 0, 2),
		values:    make([]*dfsr.Backlog, 0, size),
	}
	u.received.Add(size)
	u.started.Add(1)
	u.ended.Add(1)
	return u
}

func (u *Update) send(backlog *dfsr.Backlog) {
	u.mutex.Lock()

	if u.canceled {
		u.mutex.Unlock()
		return
	}

	if u.remaining == 0 {
		panic("backlog data is being sent to an update that has finished receiving data")
	}

	listeners := u.listeners[0:len(u.listeners)] // Snapshot of listeners

	u.values = append(u.values, backlog)
	u.received.Done()
	u.remaining--

	if u.remaining == 0 {
		// The listeners are no longer needed so let them be cleaned up by the
		// garbage collector even if the update sticks around.
		u.listeners = nil
	}

	u.mutex.Unlock()

	for i := range listeners {
		go listeners[i].send(backlog)
	}
}

func (u *Update) cancel() {
	u.mutex.Lock()

	if u.canceled || u.remaining == 0 {
		u.mutex.Unlock()
		return
	}

	listeners := u.listeners // Snapshot of listeners

	u.canceled = true
	u.listeners = nil
	u.received.Add(-u.remaining)

	u.mutex.Unlock()

	for i := range listeners {
		go listeners[i].cancel()
	}
}

func (u *Update) setStart(t time.Time) {
	u.start = t
	u.started.Done()
}

func (u *Update) setEnd(t time.Time) {
	u.end = t
	u.ended.Done()
}

type updateListener struct {
	mutex     *sync.Mutex
	ch        chan<- *dfsr.Backlog
	remaining int
}

func (ul *updateListener) init(ch chan<- *dfsr.Backlog, remaining int) {
	ul.mutex = new(sync.Mutex)
	ul.ch = ch
	ul.remaining = remaining
}

func (ul *updateListener) send(backlog *dfsr.Backlog) {
	ul.mutex.Lock()
	defer ul.mutex.Unlock()
	if ul.ch == nil {
		return // Already done or canceled
	}
	ul.ch <- backlog
	ul.remaining--
	if ul.remaining == 0 {
		close(ul.ch)
		ul.ch = nil
	}
}

func (ul *updateListener) cancel() {
	ul.mutex.Lock()
	defer ul.mutex.Unlock()
	if ul.ch == nil {
		return // Already done or canceled
	}
	close(ul.ch)
	ul.ch = nil
}
