package dfsrconfig

import (
	"time"

	"gopkg.in/dfsr.v0/dfsr"
)

// Domain will fetch DFSR configuration data from the domain.
func (c *Client) Domain() (domain dfsr.Domain, err error) {
	start := time.Now()

	nc, err := c.NamingContext()
	if err != nil {
		return
	}

	groups, err := c.Groups()
	if err != nil {
		return
	}

	return dfsr.Domain{
		NamingContext:  nc,
		Groups:         groups,
		ConfigDuration: time.Now().Sub(start),
	}, nil
}
