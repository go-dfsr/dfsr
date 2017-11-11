package dfsrconfig

import (
	"gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0/core"
)

// Domain will fetch DFSR configuration data from the specified domain using the
// provided ADSI client.
func Domain(client *adsi.Client, domain string) (data core.Domain, err error) {
	c := NewClient(client, domain)
	return c.Domain()
}

// Group will fetch DFSR configuration data for the replication group in the
// specified domain that matches the given name using the provided ADSI client.
func Group(client *adsi.Client, domain, groupName string) (data core.Group, err error) {
	c := NewClient(client, domain)
	return c.GroupByName(groupName)
}
