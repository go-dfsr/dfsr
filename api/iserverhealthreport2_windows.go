// +build windows

package api

import (
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/scjalliance/comutil"
)

// NewIServerHealthReport2 returns a new instance of the IServerHealthReport2
// component object model interface.
//
// In a typical use case, the provided clsid should CLSID_DFSRHelper
func NewIServerHealthReport2(server string, clsid *ole.GUID) (*IServerHealthReport2, error) {
	p, err := comutil.CreateRemoteObject(server, clsid, IID_IServerHealthReport2)
	return (*IServerHealthReport2)(unsafe.Pointer(p)), err
}

// GetReport retrieves a report for the given replication group.
//
// [MS-DFSRH]: 3.1.5.4.5
func (v *IServerHealthReport) GetReport(group ole.GUID, server string, referenceVectors *ole.SafeArrayConversion, flags int32) (memberVectors *ole.SafeArrayConversion, report string, err error) {
	sbstr := ole.SysAllocStringLen(server)
	if sbstr == nil {
		return nil, "", ole.NewError(ole.E_OUTOFMEMORY)
	}
	defer ole.SysFreeString(sbstr)
	var rbstr *int16
	hr, _, _ := syscall.Syscall9(
		uintptr(v.VTable().GetReport),
		3,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&group)),
		uintptr(0),
		uintptr(unsafe.Pointer(&sbstr)),
		uintptr(unsafe.Pointer(referenceVectors.Array)),
		uintptr(flags),
		uintptr(unsafe.Pointer(&memberVectors.Array)),
		uintptr(unsafe.Pointer(&rbstr)),
		0)
	if rbstr != nil {
		defer ole.SysFreeString(rbstr)
	}
	if hr == 0 {
		report = ole.BstrToString((*uint16)(unsafe.Pointer(rbstr)))
	} else {
		return nil, "", convertHresultToError(hr)
	}
	return
}
