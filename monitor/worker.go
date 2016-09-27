package monitor

import (
	"log"
	"sync"
	"time"

	"gopkg.in/dfsr.v0/callstat"
	"gopkg.in/dfsr.v0/core"
	"gopkg.in/dfsr.v0/helper"
	"gopkg.in/dfsr.v0/valuesink"
)

// worker acts as a polling source for poller. It retrieves domain configuration
// data from a configuration source, queries the DFSR backlog for all enabled
// DFSR connections in the domain and sends backlog updates via a broadcaster.
type worker struct {
	source Source
	client *helper.Client
	sink   *valuesink.Sink
	bc     *backlogBroadcaster
}

func (w *worker) Close() {
	w.client.Close()
}

func (w *worker) Poll() {
	domain, _, err := w.source.Value()
	if err != nil {
		return
	}

	conns := connections(domain)
	if len(conns) == 0 {
		return
	}

	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(len(conns))

	for _, conn := range conns {
		go w.compute(conn, &wg)
	}

	wg.Wait()

	duration := time.Now().Sub(start)

	log.Printf("Polling completed in %v.", duration)
}

func (w *worker) compute(backlog *core.Backlog, wg *sync.WaitGroup) {
	var call callstat.Call
	backlog.Backlog, call, backlog.Err = w.client.Backlog(backlog.From, backlog.To, *backlog.Group.ID)
	backlog.Duration = call.Duration()
	//w.sink.Update(backlog, timestamp, err) // TODO: Figure out a representation for value sink
	w.bc.Broadcast(backlog)
	wg.Done()
}
