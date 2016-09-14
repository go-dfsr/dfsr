package monitor

import (
	"sync"
	"time"

	"gopkg.in/dfsr.v0/core"
)

// backlogBroadcaster broadcasts backlog data to a set of listeners.
type backlogBroadcaster struct {
	mutex     sync.RWMutex
	listeners []backlogListener
	closed    bool
}

type backlogListener struct {
	c       chan *core.Backlog
	timeout time.Duration
}

func (bc *backlogBroadcaster) Close() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if bc.closed {
		return
	}
	bc.closed = true

	for _, listener := range bc.listeners {
		close(listener.c)
	}
	bc.listeners = nil
}

func (bc *backlogBroadcaster) Listen(chanSize int, timeout time.Duration) <-chan *core.Backlog {
	ch := make(chan *core.Backlog, chanSize)
	bc.mutex.Lock()
	if !bc.closed {
		bc.listeners = append(bc.listeners, backlogListener{
			c:       ch,
			timeout: timeout,
		})
	} else {
		close(ch)
	}
	bc.mutex.Unlock()
	return ch
}

func (bc *backlogBroadcaster) Unlisten(c <-chan *core.Backlog) (found bool) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	for i := 0; i < len(bc.listeners); i++ {
		entry := bc.listeners[i].c
		if entry != c {
			continue
		}

		found = true
		if i+1 < len(bc.listeners) {
			bc.listeners = append(bc.listeners[:i], bc.listeners[i+1:]...)
		} else {
			bc.listeners = bc.listeners[:i]
		}
		i--
		close(entry)
	}
	return
}

func (bc *backlogBroadcaster) Broadcast(backlog *core.Backlog) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if len(bc.listeners) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(bc.listeners))
	for _, listener := range bc.listeners {
		go sendBacklog(backlog, listener.c, listener.timeout, &wg)
	}
	wg.Wait()
}

func sendBacklog(backlog *core.Backlog, c chan<- *core.Backlog, timeout time.Duration, wg *sync.WaitGroup) {
	select {
	case c <- backlog:
	case <-time.After(timeout):
	}
	wg.Done()
}
