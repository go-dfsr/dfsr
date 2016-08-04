package api

import (
	"unsafe"

	"github.com/go-ole/go-ole"
)

// IServerHealthReportVtbl represents the component object model virtual
// function table for the IServerHealthReport interface.
type IServerHealthReportVtbl struct {
	ole.IUnknownVtbl
	GetReport                  uintptr
	GetCompressedReport        uintptr
	GetRawReportEx             uintptr
	GetReferenceVersionVectors uintptr
	_                          uintptr // Opnum7NotUsedOnWire
	GetReferenceBacklogCounts  uintptr
}

// IServerHealthReport represents the component object model interface for
// server health reports (version 1).
type IServerHealthReport struct {
	ole.IUnknown
}

// VTable returns the IServerHealthReportVtbl for the IServerHealthReport.
func (v *IServerHealthReport) VTable() *IServerHealthReportVtbl {
	return (*IServerHealthReportVtbl)(unsafe.Pointer(v.RawVTable))
}
