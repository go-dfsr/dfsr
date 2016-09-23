package helper

import (
	"sync"
	"time"

	"github.com/go-ole/go-ole"
	"gopkg.in/dfsr.v0/versionvector"
)

var _ = (Reporter)((*durableReporter)(nil)) // Compile-time interface compliance check

type reporterAction func(Reporter) error

// durableReporter provides a durable implementation of the Reporter interface
// that attempts to recreate the underlying reporter whenever an error
// occurs, and will optionally retry any failed call.
type durableReporter struct {
	server   string
	interval time.Duration
	attempts uint

	mutex        sync.RWMutex
	r            Reporter
	lastRecovery time.Time
}

// NewDurableReporter creates a durable implementation of the Reporter interface
// that is capable of recreating the underlying reporter whenever an error
// occurs and retrying any failed call.
//
// The provided interval specifies a minumum time between recovery attempts.
//
// The returned reporter will retry any failed call up to the specified number
// of retries, which may be zero. These retries will block the call until
// a successful result is returned or the maximum number of retries has been
// reached.
func NewDurableReporter(server string, interval time.Duration, retries uint) (recovering Reporter, err error) {
	r, err := NewReporter(server)
	if err != nil {
		return nil, err
	}

	return &durableReporter{
		server:       server,
		interval:     interval,
		attempts:     retries + 1,
		r:            r,
		lastRecovery: time.Now(),
	}, nil
}

func (r *durableReporter) Vector(group ole.GUID) (vector *versionvector.Vector, err error) {
	err = r.attempt(func(reporter Reporter) error {
		vector, err = reporter.Vector(group)
		return err
	})
	return
}

func (r *durableReporter) Backlog(vector *versionvector.Vector) (backlog []int, err error) {
	err = r.attempt(func(reporter Reporter) error {
		backlog, err = reporter.Backlog(vector)
		return err
	})
	return
}

func (r *durableReporter) Report(group *ole.GUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, err error) {
	err = r.attempt(func(reporter Reporter) error {
		data, report, err = reporter.Report(group, vector, backlog, files)
		return err
	})
	return
}

func (r *durableReporter) Close() {
	r.r.Close()
}

func (r *durableReporter) attempt(action reporterAction) (err error) {
	var reattempt bool
	for i := uint(0); i < r.attempts; i++ {
		r.mutex.RLock()
		reporter := r.r
		r.mutex.RUnlock()

		err = action(reporter)
		reattempt, err = r.assess(i, reporter, err)
		if !reattempt {
			return
		}
	}
	return
}

// assess assesses the error returned by the given reporter and attempts to
// restart the reporter if it's appropriate to do so.
func (r *durableReporter) assess(attempt uint, reporter Reporter, err error) (reattempt bool, resultingError error) {
	if err == nil {
		return false, err
	}
	if err == ErrClosed {
		return false, err
	}
	if attempt+1 >= r.attempts {
		// No more retries allowed, spawn a non-blocking recovery attempt
		go r.recover(reporter)
		return false, err
	}
	// Block while we attempt recovery
	if rerr := r.recover(reporter); rerr != nil {
		return true, rerr
	}
	return true, err
}

// recover attempts to recreate the underlying reporter if it is permissible.
func (r *durableReporter) recover(reporter Reporter) (err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.r != reporter {
		// Another goroutine has already performed recovery
		return
	}
	if time.Now().Sub(r.lastRecovery) < r.interval {
		// Not enough time has passed since the last recovery attempt
		return
	}
	reporter, err = NewReporter(r.server)
	if err == nil {
		go r.r.Close()
		r.r = reporter
	}
	return
}
