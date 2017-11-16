// +build !windows

package api

import (
	"github.com/go-ole/go-ole"
	"github.com/google/uuid"
)

// NewIServerHealthReport returns a new instance of the IServerHealthReport
// component object model interface.
//
// In a typical use case, the provided clsid should CLSID_DFSRHelper.
func NewIServerHealthReport(server string, clsid uuid.UUID) (*IServerHealthReport, error) {
	return nil, ole.NewError(ole.E_NOTIMPL)
}

// GetReferenceVersionVectors retrieves the version vectors for the given
// replication group. Version vectors for each replication folder in the group
// are returned.
//
// [MS-DFSRH]: 3.1.5.4.5
func (v *IServerHealthReport) GetReferenceVersionVectors(group uuid.UUID) (vectors *ole.SafeArrayConversion, err error) {
	return nil, ole.NewError(ole.E_NOTIMPL)
}

// GetReferenceBacklogCounts retrieves the number of items in the collection.
//
// [MS-DFSRH]: 3.1.5.4.5
func (v *IServerHealthReport) GetReferenceBacklogCounts(vectors *ole.SafeArrayConversion) (backlog *ole.SafeArrayConversion, err error) {
	return nil, ole.NewError(ole.E_NOTIMPL)
}
