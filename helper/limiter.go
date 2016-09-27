package helper

import (
	"github.com/go-ole/go-ole"
	"gopkg.in/dfsr.v0/callstat"
	"gopkg.in/dfsr.v0/versionvector"
)

var _ = (Reporter)((*limiter)(nil)) // Compile-time interface compliance check

// limiter provides a throttled implementation of the Reporter interface that
// wraps an underyling Reporter.
//
// limiter pushes queries onto work queues that are fed into work pools of
// a configurable number of workers. Its purpose is to limit the amount of work
// pressure that is exerted on a particular server.
type limiter struct {
	r   Reporter
	vwp *vectorWorkPool
}

// NewLimiter adds a work pool to the given Reporter. The number of workers
// is specified by numWorkers.
//
// The returned Reporter pushes queries onto work queues that are fed into work
// pools of a configurable number of workers. Its purpose is to limit the amount
// of work pressure that is exerted on a particular server.
func NewLimiter(r Reporter, numWorkers uint) (limited Reporter, err error) {
	vwp, err := newVectorWorkPool(numWorkers, r)
	if err != nil {
		return nil, err
	}

	return &limiter{
		r:   r,
		vwp: vwp,
	}, nil
}

func (l *limiter) Vector(group ole.GUID) (vector *versionvector.Vector, call callstat.Call, err error) {
	call.Begin("Limiter.Vector")
	defer call.Complete(err)
	var subcall callstat.Call
	vector, subcall, err = l.vwp.Vector(group)
	call.Add(&subcall)
	return
}

func (l *limiter) Backlog(vector *versionvector.Vector) (backlog []int, call callstat.Call, err error) {
	return l.r.Backlog(vector)
}

func (l *limiter) Report(group *ole.GUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, call callstat.Call, err error) {
	return l.r.Report(group, vector, backlog, files)
}

func (l *limiter) Close() {
	l.vwp.Close()
	l.r.Close()
}
