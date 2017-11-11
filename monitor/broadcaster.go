package monitor

import (
	"sync"

	"gopkg.in/dfsr.v0/dfsr"
)

// broadcaster broadcasts backlog updates to a set of listeners.
type broadcaster struct {
	mutex     sync.RWMutex
	listeners []chan *Update
	closed    bool
}

func (bc *broadcaster) Close() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if bc.closed {
		return
	}
	bc.closed = true

	for _, listener := range bc.listeners {
		close(listener)
	}
	bc.listeners = nil
}

func (bc *broadcaster) Listen(chanSize int) <-chan *Update {
	ch := make(chan *Update, chanSize)
	bc.mutex.Lock()
	if !bc.closed {
		bc.listeners = append(bc.listeners, ch)
	} else {
		close(ch)
	}
	bc.mutex.Unlock()
	return ch
}

func (bc *broadcaster) Unlisten(ch <-chan *Update) (found bool) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	for i := 0; i < len(bc.listeners); i++ {
		entry := bc.listeners[i]
		if entry != ch {
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

func (bc *broadcaster) Broadcast(domain *dfsr.Domain, size int) (updates []*Update) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if len(bc.listeners) == 0 {
		return
	}

	updates = make([]*Update, len(bc.listeners))

	for i, listener := range bc.listeners {
		update := newUpdate(domain, size)
		updates[i] = update
		listener <- update
	}
	return
}
