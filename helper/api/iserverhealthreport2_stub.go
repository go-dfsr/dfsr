// +build !windows

package api

import "github.com/go-ole/go-ole"

// NewIServerHealthReport2 returns a new instance of the IServerHealthReport2
// component object model interface.
//
// In a typical use case, the provided clsid should be CLSID_DFSRHelper
func NewIServerHealthReport2(server string, clsid *ole.GUID) (*IServerHealthReport, error) {
	return nil, ole.NewError(ole.E_NOTIMPL)
}

// GetReport retrieves a report for the given replication group.
//
// [MS-DFSRH]: 3.1.5.4.5
func (v *IServerHealthReport) GetReport(group ole.GUID, server string, referenceVectors ole.SafeArrayConversion, flags int32) (memberVectors ole.SafeArrayConversion, report string, err error) {
	return ole.SafeArrayConversion{}, "", ole.NewError(ole.E_NOTIMPL)
}
