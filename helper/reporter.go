package helper

import (
	"errors"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/scjalliance/comshim"

	"gopkg.in/dfsr.v0/helper/api"
	"gopkg.in/dfsr.v0/versionvector"
)

// Reporter provides access to the system API for DFSR health reports.
//
// All implementations of the Reporter interface must be threadsafe.
type Reporter interface {
	Close()
	Vector(group ole.GUID) (vector *versionvector.Vector, err error)
	Backlog(vector *versionvector.Vector) (backlog []int, err error)
	Report(group *ole.GUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, err error)
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
func (r *reporter) Vector(group ole.GUID) (vector *versionvector.Vector, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.closed() {
		return nil, ErrClosed
	}
	// TODO: Check dimensions of the returned vectors for sanity
	sa, err := r.iface.GetReferenceVersionVectors(group)
	if err != nil {
		return
	}
	return versionvector.New(sa)
}

// Backlog returns the current backlog when compared against the given
// reference version vector.
func (r *reporter) Backlog(vector *versionvector.Vector) (backlog []int, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.closed() {
		return nil, ErrClosed
	}
	// TODO: Check dimensions of the returned backlog for sanity
	sa, err := r.iface.GetReferenceBacklogCounts(vector.Data())
	if err != nil {
		return nil, err
	}

	return makeBacklog(sa), nil
}

// Report generates a report when compared against the given
// reference version vector.
func (r *reporter) Report(group *ole.GUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, err error) {
	if backlog && vector == nil {
		return nil, "", errors.New("Backlog reports require that a reference member vector is provided.")
	}

	r.m.Lock()
	defer r.m.Unlock()
	if r.closed() {
		return nil, "", ErrClosed
	}
	// TODO: Check dimensions of the returned backlog for sanity

	flags := api.REPORTING_FLAGS_NONE
	if backlog {
		flags |= api.REPORTING_FLAGS_BACKLOG
	}
	if files {
		flags |= api.REPORTING_FLAGS_FILES
	}

	if backlog {
		return r.iface.GetReport(*group, "", vector.Data(), int32(flags))
	}
	return r.iface.GetReport(*group, "", nil, int32(flags))
}
