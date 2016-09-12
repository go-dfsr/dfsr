package helper

import (
	"github.com/Jeffail/tunny"
	"github.com/go-ole/go-ole"
	"gopkg.in/dfsr.v0/versionvector"
)

type vectorWorkPool struct {
	p *tunny.WorkPool
}

func newVectorWorkPool(numWorkers uint, r Reporter) (pool *vectorWorkPool, err error) {
	if numWorkers == 0 {
		return nil, ErrZeroWorkers
	}
	workers := make([]tunny.TunnyWorker, 0, numWorkers)
	for i := uint(0); i < numWorkers; i++ {
		workers = append(workers, &vectorWorker{r: r})
	}
	p, err := tunny.CreateCustomPool(workers).Open()
	if err != nil {
		return
	}
	return &vectorWorkPool{p: p}, nil
}

func (vwp *vectorWorkPool) Vector(group ole.GUID) (vector *versionvector.Vector, err error) {
	v, err := vwp.p.SendWork(group)
	if err != nil {
		return
	}

	result, ok := v.(*vectorWorkResult)
	if !ok {
		panic("invalid work result")
	}

	return result.Vector, result.Err
}

func (vwp *vectorWorkPool) Close() {
	vwp.p.Close()
}

type vectorWorkResult struct {
	Vector *versionvector.Vector
	Err    error
}

type vectorWorker struct {
	r Reporter
}

func (vw *vectorWorker) TunnyReady() bool {
	return true
}

func (vw *vectorWorker) TunnyJob(data interface{}) interface{} {
	guid, ok := data.(ole.GUID)
	if ok {
		vector, err := vw.r.Vector(guid)
		return &vectorWorkResult{
			Vector: vector,
			Err:    err,
		}
	}
	return nil
}
