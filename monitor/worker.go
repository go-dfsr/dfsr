package monitor

import (
	"context"
	"sync"
	"time"

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
	bc     *broadcaster
}

func (w *worker) Close() {
	w.client.Close()
}

func (w *worker) Poll(ctx context.Context) {
	domain, _, err := w.source.Value()
	if err != nil {
		return
	}

	conns := connections(domain)
	if len(conns) == 0 {
		return
	}
	size := len(conns)

	updates := w.bc.Broadcast(domain, size)

	var computed, sent sync.WaitGroup
	computed.Add(size)
	sent.Add(size)

	start := time.Now()

	for _, conn := range conns {
		go w.compute(ctx, conn, updates, &computed, &sent)
	}

	for _, update := range updates {
		update.setStart(start)
	}

	computed.Wait()
	end := time.Now()

	sent.Wait()

	for _, update := range updates {
		update.setEnd(end)
	}
}

func (w *worker) compute(ctx context.Context, backlog *core.Backlog, updates []*Update, computed, sent *sync.WaitGroup) {
	if ctx == nil {
		panic("nil context")
	}
	var values []int

	values, backlog.Call, backlog.Err = w.client.Backlog(ctx, backlog.From, backlog.To, *backlog.Group.ID)
	computed.Done()

	if n := len(values); n == len(backlog.Group.Folders) {
		backlog.Folders = make([]core.FolderBacklog, n)
		for v := 0; v < n; v++ {
			backlog.Folders[v].Folder = &backlog.Group.Folders[v]
			backlog.Folders[v].Backlog = values[v]
		}
	}

	//w.sink.Update(backlog, timestamp, err) // TODO: Figure out a representation for value sink
	for _, update := range updates {
		update.send(backlog)
	}
	sent.Done()
}
