package helper

import (
	"context"
	"errors"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/scjalliance/comshim"

	"gopkg.in/dfsr.v0/callstat"
	"gopkg.in/dfsr.v0/dfsr"
	"gopkg.in/dfsr.v0/helper/api"
	"gopkg.in/dfsr.v0/versionvector"
)

type vectorResult struct {
	vector *versionvector.Vector
	err    error
}

type backlogResult struct {
	backlog []int
	err     error
}

// Reporter provides access to the system API for DFSR health reports.
//
// All implementations of the Reporter interface must be threadsafe.
type Reporter interface {
	Close()
	Vector(ctx context.Context, group ole.GUID, tracker dfsr.Tracker) (vector *versionvector.Vector, call callstat.Call, err error)
	Backlog(ctx context.Context, vector *versionvector.Vector, tracker dfsr.Tracker) (backlog []int, call callstat.Call, err error)
	Report(ctx context.Context, group *ole.GUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, call callstat.Call, err error)
}

var _ = (Reporter)((*reporter)(nil)) // Compile-time interface compliance check

// reporter provides access to the system API for DFSR health reports.
type reporter struct {
	m     sync.RWMutex
	iface *api.IServerHealthReport2
}

// NewReporter creates a new reporter. When done with a reporter it should be
// closed with a call to Close(). If New is successful it will return a reporter
// and error will be nil, otherwise the returned reporter will be nil and error
// will be non-nil.
func NewReporter(server string) (Reporter, error) {
	comshim.Add(1)
	r := &reporter{}
	err := r.init(server)
	if err != nil {
		comshim.Done()
		return nil, err
	}
	// TODO: Add finalizer for r?
	return r, nil
}

func (r *reporter) init(server string) (err error) {
	r.iface, err = api.NewIServerHealthReport2(server, api.CLSID_DFSRHelper)
	return
}

func (r *reporter) closed() bool {
	return (r.iface == nil)
}

// Close will release resources consumed by the scheduler. It should be called
// when finished with the scheduler.
func (r *reporter) Close() {
	r.m.Lock()
	defer r.m.Unlock()
	if r.closed() {
		return
	}
	r.iface.Release()
	r.iface = nil
}

// Vector returns the reference version vectors for the requested replication
// group.
//
// If the provided context is cancelled this function may return prior to the
// completion of the remote procedure call. In such a case the error returned
// will be the same as the error value of the cancelled context. The RPC call
// will consume resources in its own goroutine until the RPC call completes
// successfully or fails, at which point the result will go unreported.
//
// It is possible for goroutines to become abandoned and yet block indefinitely
// when RPC calls neither succeed nor fail. Such a situation is possible when a
// remote host is alive but unable to access its local disk due to faulting
// storage media.
//
// If a tracker is provided it will be used to signal completion of the remote
// procedure call. This can be used to gain insight into the number of
// running RPC calls and provides a means of detecting unresponsive hosts.
//
// If tracker is nil it will be ignored.
func (r *reporter) Vector(ctx context.Context, group ole.GUID, tracker dfsr.Tracker) (vector *versionvector.Vector, call callstat.Call, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	call.Begin("Reporter.Vector")
	defer call.Complete(err)

	// Early out if closed
	if r.closed() {
		err = ErrClosed
		return
	}

	// Early out if cancelled
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	// Make the call in its own goroutine
	ch := r.vector(group, tracker)

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case result := <-ch:
		vector, err = result.vector, result.err
	}

	return
}

// vector is responsible for making the low-level GetReferenceVersionVectors
// api call and returning the result on a channel. It expects the caller to
// hold a lock for the duration of the call.
//
// This call will not block, but will run the remote procedure call in its
// own goroutine.
//
// If tracker is nil it will be ignored.
func (r *reporter) vector(group ole.GUID, tracker dfsr.Tracker) <-chan vectorResult {
	ch := make(chan vectorResult, 1)
	go func() {
		defer close(ch)
		if tracker != nil {
			tc := tracker.Add()
			defer tc.Done()
		}

		// TODO: Check dimensions of the returned vectors for sanity
		sa, err := r.iface.GetReferenceVersionVectors(group)
		if err != nil {
			ch <- vectorResult{err: err}
			return
		}

		vector, err := versionvector.New(sa)
		ch <- vectorResult{vector: vector, err: err}
	}()
	return ch
}

// Backlog returns the current backlog when compared against the given
// reference version vector.
//
// If the provided context is cancelled this function may return prior to the
// completion of the remote procedure call. In such a case the error returned
// will be the same as the error value of the cancelled context. The RPC call
// will consume resources in its own goroutine until the RPC call completes
// successfully or fails, at which point the result will go unreported.
//
// It is possible for goroutines to become abandoned and yet block indefinitely
// when RPC calls neither succeed nor fail. Such a situation is possible when a
// remote host is alive but unable to access its local disk due to faulting
// storage media.
//
// If a tracker is provided it will be used to signal completion of the remote
// procedure call. This can be used to gain insight into the number of
// running RPC calls and provides a means of detecting unresponsive hosts.
//
// If tracker is nil it will be ignored.
func (r *reporter) Backlog(ctx context.Context, vector *versionvector.Vector, tracker dfsr.Tracker) (backlog []int, call callstat.Call, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	call.Begin("Reporter.Backlog")
	defer call.Complete(err)

	// Early out if closed
	if r.closed() {
		err = ErrClosed
		return
	}

	// Early out if cancelled
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	// Make the call in its own goroutine
	ch := r.backlog(vector, tracker)

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case result := <-ch:
		backlog, err = result.backlog, result.err
	}

	return
}

// backlog is responsible for making the low-level GetReferenceBacklogCounts
// api call and returning the result on a channel. It expects the caller to
// hold a lock for the duration of the call.
//
// This call will not block, but will run the remote procedure call in its
// own goroutine.
//
// If tracker is nil it will be ignored.
func (r *reporter) backlog(vector *versionvector.Vector, tracker dfsr.Tracker) <-chan backlogResult {
	ch := make(chan backlogResult, 1)
	go func() {
		defer close(ch)
		if tracker != nil {
			tc := tracker.Add()
			defer tc.Done()
		}

		// TODO: Check dimensions of the returned vectors for sanity
		sa, err := r.iface.GetReferenceBacklogCounts(vector.Data())
		if err != nil {
			ch <- backlogResult{err: err}
			return
		}
		defer sa.Release()

		ch <- backlogResult{backlog: makeBacklog(sa)}
	}()
	return ch
}

// Report generates a report when compared against the reference version vector.
func (r *reporter) Report(ctx context.Context, group *ole.GUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, call callstat.Call, err error) {
	if backlog && vector == nil {
		call.Description = "Reporter.Report"
		err = errors.New("Backlog reports require that a reference member vector is provided.")
		call.Complete(err)
		return
	}

	r.m.Lock()
	defer r.m.Unlock()
	call.Begin("Reporter.Report")
	defer call.Complete(err)

	if r.closed() {
		err = ErrClosed
		return
	}

	// Handle cancellation
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	flags := api.REPORTING_FLAGS_NONE
	if backlog {
		flags |= api.REPORTING_FLAGS_BACKLOG
	}
	if files {
		flags |= api.REPORTING_FLAGS_FILES
	}

	var vdata *ole.SafeArrayConversion
	if backlog {
		vdata = vector.Data()
	}

	// TODO: Check dimensions of the returned backlog for sanity
	data, report, err = r.iface.GetReport(*group, "", vdata, int32(flags))
	return
}
