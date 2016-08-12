package versionvector

import (
	"github.com/go-ole/go-ole"
	"github.com/scjalliance/comutil"
)

// Vector represents version vector data from a replication group member.
type Vector struct {
	sa *ole.SafeArrayConversion
}

// New returns a new version vector for the given safe array of data.
func New(sa *ole.SafeArrayConversion) (vector *Vector, err error) {
	return &Vector{
		sa: sa,
	}, nil
}

// Data returns the version vector data as a safe array.
func (vector *Vector) Data() (sa *ole.SafeArrayConversion) {
	return vector.sa
}

// Duplicate will return a duplicate of the vector that does not share any
// memory with the original.
func (vector *Vector) Duplicate() (duplicate *Vector, err error) {
	sa, err := comutil.SafeArrayCopy(vector.Data().Array)
	if err != nil {
		return
	}

	return New(&ole.SafeArrayConversion{Array: sa})
}

// Close will release any resources consumed by the version vector. It should be
// called when finished with the version vector.
func (vector *Vector) Close() {
	if vector.sa != nil {
		vector.sa.Release()
		vector.sa = nil
	}
}
