package config

import (
	"github.com/go-ole/go-ole"
	"gopkg.in/adsi.v0"
	"gopkg.in/dfsr.v0"
)

// Domain will fetch DFSR configuration data from the specified domain using the
// provided ADSI client.
func Domain(client *adsi.Client, domain string) (data dfsr.Domain, err error) {
	fetch, err := newFetcher(client, domain)
	if err != nil {
		return
	}
	defer fetch.Close()

	return fetch.Domain()
}

// Group will fetch DFSR configuration data for the replication group in the
// specified domain that matches the given name using the provided ADSI client.
func Group(client *adsi.Client, domain, groupName string) (data dfsr.Group, err error) {
	fetch, err := newFetcher(client, domain)
	if err != nil {
		return
	}
	defer fetch.Close()

	return fetch.GroupByName(groupName)
}

// GroupByGUID will fetch DFSR configuration data for the requested replication
// group in the specified domain using the provided ADSI client.
func GroupByGUID(client *adsi.Client, domain string, group *ole.GUID) (data dfsr.Group, err error) {
	fetch, err := newFetcher(client, domain)
	if err != nil {
		return
	}
	defer fetch.Close()

	return fetch.GroupByGUID(group)
}
