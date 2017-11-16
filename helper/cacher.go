package helper

import (
	"context"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/google/uuid"
	"gopkg.in/dfsr.v0/callstat"
	"gopkg.in/dfsr.v0/dfsr"
	"gopkg.in/dfsr.v0/versionvector"
)

var _ = (Reporter)((*cacher)(nil)) // Compile-time interface compliance check

// cacher provides a caching implementation of the Reporter interface that wraps
// an underyling Reporter.
type cacher struct {
	r  Reporter
	vc *versionvector.Cache
}

// NewCacher adds an expiring vector cache to the given Reporter. The duration
// of cached values is specified by duration.
func NewCacher(r Reporter, duration time.Duration) (cached Reporter) {
	return &cacher{
		r:  r,
		vc: versionvector.NewCache(duration, r.Vector),
	}
}

// FIXME: Make the underlying vector cache handle contexts from mulitple pending callers.
func (c *cacher) Vector(ctx context.Context, group uuid.UUID, tracker dfsr.Tracker) (vector *versionvector.Vector, call callstat.Call, err error) {
	return c.vc.Lookup(ctx, group, tracker)
}

func (c *cacher) Backlog(ctx context.Context, vector *versionvector.Vector, tracker dfsr.Tracker) (backlog []int, call callstat.Call, err error) {
	return c.r.Backlog(ctx, vector, tracker)
}

func (c *cacher) Report(ctx context.Context, group uuid.UUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, call callstat.Call, err error) {
	return c.r.Report(ctx, group, vector, backlog, files)
}

func (c *cacher) Close() {
	c.vc.Close()
	c.r.Close()
}
