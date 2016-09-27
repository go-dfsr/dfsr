package helper

import (
	"time"

	"github.com/go-ole/go-ole"
	"gopkg.in/dfsr.v0/callstat"
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

func (c *cacher) Vector(group ole.GUID) (vector *versionvector.Vector, call callstat.Call, err error) {
	return c.vc.Lookup(group)
}

func (c *cacher) Backlog(vector *versionvector.Vector) (backlog []int, call callstat.Call, err error) {
	return c.r.Backlog(vector)
}

func (c *cacher) Report(group *ole.GUID, vector *versionvector.Vector, backlog, files bool) (data *ole.SafeArrayConversion, report string, call callstat.Call, err error) {
	return c.r.Report(group, vector, backlog, files)
}

func (c *cacher) Close() {
	c.vc.Close()
	c.r.Close()
}
