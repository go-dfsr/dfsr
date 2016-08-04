package api

type DfsrReportingFlags int

const (
	REPORTING_FLAGS_NONE DfsrReportingFlags = iota
	REPORTING_FLAGS_BACKLOG
	REPORTING_FLAGS_FILES
)

type DfsrHelperErrorsEnum int

const (
	dfsrHelperErrorNotLocalAdmin             DfsrHelperErrorsEnum = 0x80042001
	dfsrHelperErrorCreateVerifyServerControl DfsrHelperErrorsEnum = 0x80042002
	dfsrHelperLdapErrorBase                  DfsrHelperErrorsEnum = 0x80043000
)
