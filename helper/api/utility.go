package api

import (
	"strings"

	"github.com/go-ole/go-ole"
)

// convertHresultToError converts syscall to error, if call is unsuccessful.
func convertHresultToError(hr uintptr) (err error) {
	if hr != 0 {
		err = ole.NewError(hr)
		if strings.Contains(err.Error(), "FormatMessage failed") {
			switch hr {
			case E_INVALID_NAMESPACE:
				err = ErrInvalidNamespace
			case E_ACCESS_DENIED:
				err = ErrAccessDenied
			}
		}
	}
	return
}
