// +build windows

package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/gentlemanautomaton/bindflag"
)

// Settings represents a set of DFSR monitor service configuration settings
type Settings struct {
	Domain                 string
	ConfigPollingInterval  time.Duration
	ConfigPollingTimeout   time.Duration
	BacklogPollingInterval time.Duration
	BacklogPollingTimeout  time.Duration
	VectorCacheDuration    time.Duration
	Limit                  uint
	StatHatKey             string
	StatHatFormat          string
}

// DefaultSettings is the default set of DFSR monitor settings.
var DefaultSettings = Settings{
	ConfigPollingInterval:  15 * time.Minute,
	ConfigPollingTimeout:   2 * time.Minute,
	BacklogPollingInterval: 5 * time.Minute,
	BacklogPollingTimeout:  5 * time.Minute,
	VectorCacheDuration:    30 * time.Second,
	Limit:                  1,
}

// Bind will link the settings to the provided flag set.
func (s *Settings) Bind(fs *flag.FlagSet) {
	fs.Var(bindflag.String(&s.Domain), "domain", "AD domain to monitor (will autodetect if not provided)")
	fs.Var(bindflag.Duration(&s.ConfigPollingInterval), "cpi", "configuration polling interval")
	fs.Var(bindflag.Duration(&s.ConfigPollingTimeout), "cpt", "configuration polling timeout")
	fs.Var(bindflag.Duration(&s.BacklogPollingInterval), "bpi", "backlog polling interval")
	fs.Var(bindflag.Duration(&s.BacklogPollingTimeout), "bpt", "backlog polling timeout")
	fs.Var(bindflag.Duration(&s.VectorCacheDuration), "cache", "vector cache duration")
	fs.Var(bindflag.Uint(&s.Limit), "limit", "maximum number of queries per server")
	fs.Var(bindflag.String(&s.StatHatKey), "shk", "StatHat ezkey for StatHat reporting")
	fs.Var(bindflag.String(&s.StatHatFormat), "shf", "StatHat name format in fmt style")
}

// Parse parses the given argument list and applies the specified values.
func (s *Settings) Parse(args []string, errorHandling flag.ErrorHandling) (err error) {
	fs := flag.NewFlagSet("", errorHandling)
	s.Bind(fs)
	return fs.Parse(args)
}

// Args returns the current settings as a set of command line arguments that can
// be passed back into the service.
func (s *Settings) Args() (args []string) {
	if s.Domain != "" {
		args = append(args, makeArg("domain", s.Domain))
	}
	if s.ConfigPollingInterval != time.Duration(0) {
		args = append(args, makeArg("cpi", s.ConfigPollingInterval.String()))
	}
	if s.ConfigPollingTimeout != time.Duration(0) {
		args = append(args, makeArg("cpt", s.ConfigPollingTimeout.String()))
	}
	if s.BacklogPollingInterval != time.Duration(0) {
		args = append(args, makeArg("bpi", s.BacklogPollingInterval.String()))
	}
	if s.BacklogPollingTimeout != time.Duration(0) {
		args = append(args, makeArg("bpt", s.BacklogPollingTimeout.String()))
	}
	if s.VectorCacheDuration != time.Duration(0) {
		args = append(args, makeArg("cache", s.VectorCacheDuration.String()))
	}
	if s.Limit != 0 {
		args = append(args, makeArg("limit", fmt.Sprintf("%v", s.Limit)))
	}
	if s.StatHatKey != "" {
		args = append(args, makeArg("shk", s.StatHatKey))
	}
	if s.StatHatFormat != "" {
		args = append(args, makeArg("shf", s.StatHatFormat))
	}
	return
}
