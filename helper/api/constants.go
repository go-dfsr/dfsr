package api

import "errors"

const (
	E_INVALID_NAMESPACE = 2147749902
	E_ACCESS_DENIED     = 2147749891
)

var (
	ErrInvalidNamespace = errors.New("the provided name or namespace is invalid")
	ErrAccessDenied     = errors.New("access denied")
)
