// +build windows

package api

import (
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/scjalliance/comutil"
)

// NewIServerHealthReport returns a new instance of the IServerHealthReport
// component object model interface.
//
// In a typical use case, the provided clsid should CLSID_DFSRHelper
func NewIServerHealthReport(server string, clsid *ole.GUID) (*IServerHealthReport, error) {
	p, err := comutil.CreateRemoteObject(server, clsid, IID_IServerHealthReport)
	return (*IServerHealthReport)(unsafe.Pointer(p)), err
}

// GetReferenceVersionVectors retrieves the version vectors for the given
// replication group. Version vectors for each replication folder in the group
// are returned.
//
// [MS-DFSRH]: 3.1.5.4.5
func (v *IServerHealthReport) GetReferenceVersionVectors(group ole.GUID) (vectors *ole.SafeArrayConversion, err error) {
	vectors = new(ole.SafeArrayConversion)
	hr, _, _ := syscall.Syscall(
		uintptr(v.VTable().GetReferenceVersionVectors),
		3,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&group)),
		uintptr(unsafe.Pointer(&vectors.Array)))
	if hr != 0 {
		return nil, convertHresultToError(hr)
	}
	return
}

// GetReferenceBacklogCounts retrieves the number of items in the collection.
//
// [MS-DFSRH]: 3.1.5.4.5
func (v *IServerHealthReport) GetReferenceBacklogCounts(vectors *ole.SafeArrayConversion) (backlog *ole.SafeArrayConversion, err error) {
	backlog = new(ole.SafeArrayConversion)
	hr, _, _ := syscall.Syscall(
		uintptr(v.VTable().GetReferenceBacklogCounts),
		3,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&vectors.Array)),
		uintptr(unsafe.Pointer(&backlog.Array)))
	if hr != 0 {
		return nil, convertHresultToError(hr)
	}
	return
}
