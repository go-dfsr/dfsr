package monitor

import (
	"sync"

	"gopkg.in/dfsr.v0/core"
)

// backlogBroadcaster broadcasts backlog data to a set of listeners.
type backlogBroadcaster struct {
	mutex     sync.RWMutex
	listeners []chan<- *core.Backlog
	closed    bool
}

func (bc *backlogBroadcaster) Close() {
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

func (bc *backlogBroadcaster) Listen() <-chan *core.Backlog {
	ch := make(chan *core.Backlog, updateChanSize)
	bc.mutex.Lock()
	if !bc.closed {
		bc.listeners = append(bc.listeners, ch)
	} else {
		close(ch)
	}
	bc.mutex.Unlock()
	return ch
}

func (bc *backlogBroadcaster) Broadcast(backlog *core.Backlog) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if len(bc.listeners) == 0 {
		return
	}

	for _, listener := range bc.listeners {
		listener <- backlog
	}
}
