package dfsr

import (
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/scjalliance/comshim"

	"gopkg.in/dfsr.v0/api"
)

// Reporter provides access to the system API for DFSR health reports.
type Reporter struct {
	m     sync.RWMutex
	iface *api.IServerHealthReport2
}

// NewReporter creates a new reporter. When done with a reporter it should be
// closed with a call to Close(). If New is successful it will return a reporter
// and error will be nil, otherwise the returned reporter will be nil and error
// will be non-nil.
func NewReporter(server string) (*Reporter, error) {
	comshim.Add(1)
	r := &Reporter{}
	err := r.init(server)
	if err != nil {
		comshim.Done()
		return nil, err
	}
	// TODO: Add finalizer for r?
	return r, nil
}

func (r *Reporter) init(server string) (err error) {
	r.iface, err = api.NewIServerHealthReport2(server, api.CLSID_DFSRHelper)
	return
}

func (r *Reporter) closed() bool {
	return (r.iface == nil)
}

// Close will release resources consumed by the scheduler. It should be called
// when finished with the scheduler.
func (r *Reporter) Close() {
	r.m.Lock()
	defer r.m.Unlock()
	if r.closed() {
		return
	}
	r.iface.Release()
	r.iface = nil
}

// Vectors returns the reference version vectors for the requested replication
// group.
func (r *Reporter) Vectors(group *ole.GUID) (vectors *ole.SafeArrayConversion, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.closed() {
		return nil, ErrClosed
	}
	// TODO: Check dimensions of the returned vectors for sanity
	return r.iface.GetReferenceVersionVectors(*group)
}

// Backlog returns the current backlog when compared against the given
// reference version vectors.
func (r *Reporter) Backlog(vectors *ole.SafeArrayConversion) (backlog *ole.SafeArrayConversion, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.closed() {
		return nil, ErrClosed
	}
	// TODO: Check dimensions of the returned backlog for sanity
	return r.iface.GetReferenceBacklogCounts(vectors)
}
