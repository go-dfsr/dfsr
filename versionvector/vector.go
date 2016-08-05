package versionvector

import "github.com/go-ole/go-ole"

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

// Close will release any resources consumed by the version vector. It should be
// called when finished with the version vector.
func (vector *Vector) Close() {
	if vector.sa != nil {
		vector.sa.Release()
		vector.sa = nil
	}
}
