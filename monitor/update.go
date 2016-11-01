package monitor

import (
	"sync"
	"time"

	"gopkg.in/dfsr.v0/core"
)

// Update represents an in-progress DFSR update performed by the monitor.
type Update struct {
	Domain *core.Domain // Domain configuration at the time of the update
	//Connections []Connection

	size     int // Size is the number of backlog entries to be received in this update
	start    time.Time
	end      time.Time
	received sync.WaitGroup // Have we received all of the backlog values?
	started  sync.WaitGroup // Have we received the start time?
	ended    sync.WaitGroup // Have we received the end time?

	mutex     sync.Mutex
	listeners []updateListener
	values    []*core.Backlog
}

// Listen returns a channel that receives the DFSR backlog values as they are
// retrieved for this update. The values are returned in the order that they
// are received.
//
// The returned channel will be buffered with a capacity that matches the number
// of values that will be returned in this update.
func (u *Update) Listen() <-chan *core.Backlog {
	ch := make(chan *core.Backlog, u.size)
	if u.size == 0 {
		close(ch)
		return ch
	}

	u.mutex.Lock()
	e := len(u.listeners)
	u.listeners = append(u.listeners, updateListener{})
	u.listeners[e].init(ch, u.size)
	values := u.values[0:len(u.values)] // Snapshot of values
	u.mutex.Unlock()

	// Send the values that we've already received
	go func() {
		for _, v := range values {
			ch <- v
		}
	}()
	return ch
}

// Values will block until the update has finished, then return a slice of the
// retrieved backlog data.
func (u *Update) Values() (values []*core.Backlog) {
	u.received.Wait()
	return u.values
}

// Size returns the number of values that will be returned in the update.
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

func newUpdate(domain *core.Domain, size int) *Update {
	u := &Update{
		Domain:    domain,
		size:      size,
		listeners: make([]updateListener, 0, 2),
		values:    make([]*core.Backlog, 0, size),
	}
	u.received.Add(size)
	u.started.Add(1)
	u.ended.Add(1)
	return u
}

func (u *Update) send(backlog *core.Backlog) {
	u.mutex.Lock()
	u.values = append(u.values, backlog)
	listeners := u.listeners[0:len(u.listeners)] // Snapshot of listeners
	u.mutex.Unlock()

	u.received.Done()

	go func() {
		for i := range listeners {
			listener := &listeners[i]
			listener.ch <- backlog
			listener.wg.Done()
		}
	}()
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
	ch chan<- *core.Backlog
	wg sync.WaitGroup
}

func (ue *updateListener) init(ch chan<- *core.Backlog, size int) {
	ue.ch = ch
	ue.wg.Add(size)
	go func() {
		ue.wg.Wait()
		close(ch)
	}()
}
