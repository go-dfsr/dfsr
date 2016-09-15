package main

import (
	"flag"
	"log"
	"time"
)

// Settings represents a set of DFSR monitor service configuration settings
type Settings struct {
	Domain                 string
	ConfigPollingInterval  time.Duration
	BacklogPollingInterval time.Duration
	VectorCacheDuration    time.Duration
	Limit                  uint
	StatHatKey             string
	StatHatFormat          string
}

// DefaultSettings is the default set of DFSR monitor settings.
var DefaultSettings = Settings{
	ConfigPollingInterval:  15 * time.Minute,
	BacklogPollingInterval: 5 * time.Minute,
	VectorCacheDuration:    30 * time.Second,
	Limit:                  1,
}

// Parse parses the given argument list and applies the specified values.
func (s *Settings) Parse(args []string) error {
	log.Printf("Settings parsing args: %v", args)

	var fs flag.FlagSet
	cpi := fs.Uint("cpi", 0, "configuration polling interval in seconds")
	bpi := fs.Uint("bpi", 0, "backlog polling interval in seconds")
	cache := fs.Uint("cache", 0, "vector cache duration in seconds")
	limit := fs.Uint("limit", 0, "maximum number of queries per server")
	stathatkey := fs.String("shk", "", "")
	stathatformat := fs.String("shf", "", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *cpi != 0 {
		s.ConfigPollingInterval = time.Duration(*cpi) * time.Second
	}
	if *bpi != 0 {
		s.BacklogPollingInterval = time.Duration(*bpi) * time.Second
	}
	if *cache != 0 {
		s.VectorCacheDuration = time.Duration(*cache) * time.Second
	}
	if *limit != 0 {
		s.Limit = *limit
	}
	if *stathatkey != "" {
		s.StatHatKey = *stathatkey
	}
	if *stathatformat != "" {
		s.StatHatFormat = *stathatformat
	}
	log.Printf("Settings: %+v", s)
	return nil
}
