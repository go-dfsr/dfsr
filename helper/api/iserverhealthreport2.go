package api

import "unsafe"

// IServerHealthReport2Vtbl represents the component object model virtual
// function table for the IServerHealthReport2 interface.
type IServerHealthReport2Vtbl struct {
	IServerHealthReport
	GetReport           uintptr
	GetCompressedReport uintptr
}

// IServerHealthReport2 represents the component object model interface for
// server health reports (version 2).
type IServerHealthReport2 struct {
	IServerHealthReport
}

// VTable returns the IServerHealthReport2Vtbl for the IServerHealthReport2.
func (v *IServerHealthReport2) VTable() *IServerHealthReport2Vtbl {
	return (*IServerHealthReport2Vtbl)(unsafe.Pointer(v.RawVTable))
}
